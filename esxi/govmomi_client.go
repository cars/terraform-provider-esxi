package esxi

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

// GovmomiClient wraps govmomi client with provider-specific logic
type GovmomiClient struct {
	Client     *govmomi.Client
	Finder     *find.Finder
	Datacenter *object.Datacenter
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewGovmomiClient creates a new govmomi client connection
func NewGovmomiClient(config *Config) (*GovmomiClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Build connection URL
	var u *url.URL
	var err error

	// Check if esxiHostName is already a full URL (for testing with simulator)
	if u, err = url.Parse(config.esxiHostName); err == nil && u.Scheme != "" {
		// Already a full URL, just add credentials
		u.User = url.UserPassword(config.esxiUserName, config.esxiPassword)
	} else {
		// Build URL from components
		u, err = url.Parse(fmt.Sprintf("https://%s:%s/sdk",
			config.esxiHostName,
			config.esxiHostSSLport))
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to parse ESXi URL: %w", err)
		}
		u.User = url.UserPassword(config.esxiUserName, config.esxiPassword)
	}

	// Create soap client with insecure flag (ESXi often uses self-signed certs)
	soapClient := soap.NewClient(u, true)

	// Set timeout for operations
	soapClient.Timeout = 30 * time.Minute

	// Create vim25 client
	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create vim25 client: %w", err)
	}

	// Create govmomi client
	client := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	// Login
	err = client.Login(ctx, u.User)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to login to ESXi: %w", err)
	}

	// Create finder for object lookups
	finder := find.NewFinder(client.Client, true)

	// For standalone ESXi, get default datacenter (ha-datacenter)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		client.Logout(ctx)
		cancel()
		return nil, fmt.Errorf("failed to find datacenter: %w", err)
	}
	finder.SetDatacenter(dc)

	return &GovmomiClient{
		Client:     client,
		Finder:     finder,
		Datacenter: dc,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Close terminates the govmomi client session
func (gc *GovmomiClient) Close() error {
	if gc.Client != nil {
		err := gc.Client.Logout(gc.ctx)
		if gc.cancel != nil {
			gc.cancel()
		}
		return err
	}
	if gc.cancel != nil {
		gc.cancel()
	}
	return nil
}

// Context returns the client context
func (gc *GovmomiClient) Context() context.Context {
	return gc.ctx
}

// IsActive checks if the session is still active
func (gc *GovmomiClient) IsActive() (bool, error) {
	if gc.Client == nil {
		return false, nil
	}

	sessionManager := session.NewManager(gc.Client.Client)
	userSession, err := sessionManager.UserSession(gc.ctx)
	if err != nil {
		return false, err
	}

	return userSession != nil, nil
}

// Reconnect attempts to reconnect if the session is inactive
func (gc *GovmomiClient) Reconnect(config *Config) error {
	active, err := gc.IsActive()
	if err != nil || !active {
		// Close existing connection
		gc.Close()

		// Create new connection
		newClient, err := NewGovmomiClient(config)
		if err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}

		// Replace with new client
		*gc = *newClient
	}
	return nil
}
