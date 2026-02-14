package esxi

import (
	"context"
	"testing"
	"time"

	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25/types"
)

// TestGovmomiClientConnection tests basic connection to vcsim
func TestGovmomiClientConnection(t *testing.T) {
	// Create vcsim model
	model := simulator.VPX()
	model.Host = 1 // Single ESXi host

	// Start simulator
	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	// Create config from simulator
	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	// Test connection
	client, err := NewGovmomiClient(config)
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer client.Close()

	// Verify client is active
	active, err := client.IsActive()
	if err != nil {
		t.Fatalf("Failed to check if session is active: %v", err)
	}

	if !active {
		t.Fatal("Session should be active")
	}

	// Verify datacenter exists
	if client.Datacenter == nil {
		t.Fatal("Datacenter should not be nil")
	}
}

// TestGovmomiClientFindVM tests VM finder functionality
func TestGovmomiClientFindVM(t *testing.T) {
	// Create vcsim model
	model := simulator.VPX()
	model.Host = 1

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	client, err := NewGovmomiClient(config)
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer client.Close()

	// List VMs (simulator creates default VMs)
	vms, err := client.Finder.VirtualMachineList(client.Context(), "*")
	if err != nil {
		t.Fatalf("Failed to list VMs: %v", err)
	}

	if len(vms) == 0 {
		t.Fatal("Expected at least one VM in simulator")
	}

	// Try to find first VM by name
	vmName := vms[0].Name()
	vm, err := getVMByName(client.Context(), client.Finder, vmName)
	if err != nil {
		t.Fatalf("Failed to find VM by name: %v", err)
	}

	if vm == nil {
		t.Fatal("VM should not be nil")
	}
}

// TestGovmomiClientPowerState tests power state retrieval
func TestGovmomiClientPowerState(t *testing.T) {
	model := simulator.VPX()
	model.Host = 1

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	client, err := NewGovmomiClient(config)
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer client.Close()

	// Get first VM
	vms, err := client.Finder.VirtualMachineList(client.Context(), "*")
	if err != nil || len(vms) == 0 {
		t.Fatal("Failed to get VMs")
	}

	vm := vms[0]

	// Get power state
	powerState, err := getPowerState(client.Context(), vm)
	if err != nil {
		t.Fatalf("Failed to get power state: %v", err)
	}

	// Verify it's a valid power state
	validStates := []types.VirtualMachinePowerState{
		types.VirtualMachinePowerStatePoweredOn,
		types.VirtualMachinePowerStatePoweredOff,
		types.VirtualMachinePowerStateSuspended,
	}

	found := false
	for _, state := range validStates {
		if powerState == state {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Invalid power state: %v", powerState)
	}
}

// TestGovmomiClientReconnect tests session reconnection
func TestGovmomiClientReconnect(t *testing.T) {
	model := simulator.VPX()
	model.Host = 1

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	client, err := NewGovmomiClient(config)
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer client.Close()

	// Verify initial session is active
	active, err := client.IsActive()
	if err != nil || !active {
		t.Fatal("Initial session should be active")
	}

	// Test reconnect (should be a no-op if session is active)
	err = client.Reconnect(config)
	if err != nil {
		t.Fatalf("Reconnect failed: %v", err)
	}

	// Verify session is still active
	active, err = client.IsActive()
	if err != nil || !active {
		t.Fatal("Session should still be active after reconnect")
	}
}

// TestGovmomiClientContext tests context handling
func TestGovmomiClientContext(t *testing.T) {
	model := simulator.VPX()
	model.Host = 1

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	client, err := NewGovmomiClient(config)
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer client.Close()

	// Get context
	ctx := client.Context()
	if ctx == nil {
		t.Fatal("Context should not be nil")
	}

	// Verify context works with a timeout operation
	_, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try a simple operation with context
	_, err = client.Finder.VirtualMachineList(ctx, "*")
	if err != nil {
		t.Fatalf("Failed to use context: %v", err)
	}
}

// TestGovmomiClientClose tests proper cleanup
func TestGovmomiClientClose(t *testing.T) {
	model := simulator.VPX()
	model.Host = 1

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	client, err := NewGovmomiClient(config)
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}

	// Close client
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}

	// Verify session is no longer active
	active, _ := client.IsActive()
	if active {
		t.Fatal("Session should not be active after close")
	}
}

// TestConfigGetGovmomiClient tests Config's govmomi client caching
func TestConfigGetGovmomiClient(t *testing.T) {
	model := simulator.VPX()
	model.Host = 1

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	password, _ := simulator.DefaultLogin.Password()
	config := &Config{
		esxiHostName:    s.URL.String(),
		esxiHostSSLport: "443",
		esxiUserName:    simulator.DefaultLogin.Username(),
		esxiPassword:    password,
	}

	// First call should create client
	client1, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}

	if client1 == nil {
		t.Fatal("Client should not be nil")
	}

	// Second call should return cached client
	client2, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get cached govmomi client: %v", err)
	}

	if client2 == nil {
		t.Fatal("Cached client should not be nil")
	}

	// Close client
	err = config.CloseGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to close govmomi client: %v", err)
	}

	// Verify config's client is nil
	if config.govmomiClient != nil {
		t.Fatal("Config's govmomi client should be nil after close")
	}
}
