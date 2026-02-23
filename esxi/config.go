package esxi

import (
	"fmt"
	"log"
)

type Config struct {
	esxiHostName    string
	esxiHostSSHport string
	esxiHostSSLport string
	esxiUserName    string
	esxiPassword    string
	esxiPrivateKeyPath string

	// govmomi client
	govmomiClient *GovmomiClient  // Cached client connection
}

// GetGovmomiClient returns cached client or creates new one
func (c *Config) GetGovmomiClient() (*GovmomiClient, error) {
	if c.govmomiClient == nil {
		client, err := NewGovmomiClient(c)
		if err != nil {
			return nil, err
		}
		c.govmomiClient = client
	} else {
		// Check if session is still active, reconnect if needed
		err := c.govmomiClient.Reconnect(c)
		if err != nil {
			return nil, err
		}
	}
	return c.govmomiClient, nil
}

// CloseGovmomiClient closes the cached govmomi client
func (c *Config) CloseGovmomiClient() error {
	if c.govmomiClient != nil {
		err := c.govmomiClient.Close()
		c.govmomiClient = nil
		return err
	}
	return nil
}

func (c *Config) validateEsxiCreds() error {
	esxiConnInfo := getConnectionInfo(c)
	log.Printf("[validateEsxiCreds]\n")

	var remote_cmd string
	var err error

	remote_cmd = fmt.Sprintf("vmware --version")
	_, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Connectivity test, get vmware version")
	if err != nil {
		return fmt.Errorf("Failed to connect to esxi host: %s\n", err)
	}

	runRemoteSshCommand(esxiConnInfo, "mkdir -p ~", "Create home directory if missing")

	return nil
}
