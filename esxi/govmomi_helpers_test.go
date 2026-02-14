package esxi

import (
	"testing"

	"github.com/vmware/govmomi/simulator"
)

// TestGetHostSystem tests getting the default host system
func TestGetHostSystem(t *testing.T) {
	model := simulator.ESX()

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

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Get host system
	host, err := getHostSystem(client.Context(), client.Finder)
	if err != nil {
		t.Fatalf("Failed to get host system: %v", err)
	}

	if host == nil {
		t.Fatal("Host system should not be nil")
	}

	// Verify host name
	hostName := host.Name()
	if hostName == "" {
		t.Fatal("Host name should not be empty")
	}
}

// TestGetDatastoreByName tests finding a datastore by name
func TestGetDatastoreByName(t *testing.T) {
	model := simulator.ESX()

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

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// List datastores first
	datastores, err := client.Finder.DatastoreList(client.Context(), "*")
	if err != nil || len(datastores) == 0 {
		t.Fatal("Failed to get datastores or no datastores found")
	}

	// Get first datastore by name
	dsName := datastores[0].Name()
	ds, err := getDatastoreByName(client.Context(), client.Finder, dsName)
	if err != nil {
		t.Fatalf("Failed to get datastore by name: %v", err)
	}

	if ds == nil {
		t.Fatal("Datastore should not be nil")
	}

	if ds.Name() != dsName {
		t.Fatalf("Expected datastore name %s, got %s", dsName, ds.Name())
	}
}

// TestIsDatastoreAccessible tests checking datastore accessibility
func TestIsDatastoreAccessible(t *testing.T) {
	model := simulator.ESX()

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

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Get first datastore
	datastores, err := client.Finder.DatastoreList(client.Context(), "*")
	if err != nil || len(datastores) == 0 {
		t.Fatal("Failed to get datastores")
	}

	ds := datastores[0]

	// Check accessibility
	accessible, err := isDatastoreAccessible(client.Context(), ds)
	if err != nil {
		t.Fatalf("Failed to check datastore accessibility: %v", err)
	}

	if !accessible {
		t.Fatal("Datastore should be accessible in simulator")
	}
}

// TestGetHostNetworkSystem tests getting the host network system
func TestGetHostNetworkSystem(t *testing.T) {
	model := simulator.ESX()

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

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Get host system
	host, err := getHostSystem(client.Context(), client.Finder)
	if err != nil {
		t.Fatalf("Failed to get host system: %v", err)
	}

	// Get network system
	ns, err := getHostNetworkSystem(client.Context(), host)
	if err != nil {
		t.Fatalf("Failed to get network system: %v", err)
	}

	if ns == nil {
		t.Fatal("Network system should not be nil")
	}
}

// TestGetRootResourcePool tests getting the root resource pool
func TestGetRootResourcePool(t *testing.T) {
	model := simulator.ESX()

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

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Get root resource pool
	pool, err := getRootResourcePool(client.Context(), client.Finder)
	if err != nil {
		t.Fatalf("Failed to get root resource pool: %v", err)
	}

	if pool == nil {
		t.Fatal("Root resource pool should not be nil")
	}

	// Verify it's a resource pool
	poolName := pool.Name()
	if poolName == "" {
		t.Fatal("Resource pool name should not be empty")
	}
}

// TestWaitForTask tests the task waiting helper
func TestWaitForTask(t *testing.T) {
	model := simulator.ESX()

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

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Get a VM to perform an operation
	vms, err := client.Finder.VirtualMachineList(client.Context(), "*")
	if err != nil || len(vms) == 0 {
		t.Fatal("Failed to get VMs")
	}

	vm := vms[0]

	// Power off VM (if on) to test task waiting
	powerState, _ := getPowerState(client.Context(), vm)
	if powerState == "poweredOn" {
		task, err := vm.PowerOff(client.Context())
		if err != nil {
			t.Fatalf("Failed to start power off task: %v", err)
		}

		// Test waitForTask
		err = waitForTask(client.Context(), task)
		if err != nil {
			t.Fatalf("waitForTask failed: %v", err)
		}
	}
}
