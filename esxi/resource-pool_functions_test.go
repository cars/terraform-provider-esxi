package esxi

import (
	"testing"

	"github.com/vmware/govmomi/simulator"
)

// TestResourcePoolCreateReadDeleteGovmomi tests resource pool lifecycle
func TestResourcePoolCreateReadDeleteGovmomi(t *testing.T) {
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
		useGovmomi:      true,
	}

	_, err = config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Test parameters
	poolName := "test-pool"
	cpuMin := 100
	cpuMinExpandable := "true"
	cpuMax := 200
	cpuShares := "normal"
	memMin := 256
	memMinExpandable := "true"
	memMax := 512
	memShares := "normal"
	parentPool := ""

	// Create resource pool
	poolID, err := resourcePoolCreate_govmomi(config, poolName, cpuMin, cpuMinExpandable,
		cpuMax, cpuShares, memMin, memMinExpandable, memMax, memShares, parentPool)
	if err != nil {
		t.Fatalf("Failed to create resource pool: %v", err)
	}

	if poolID == "" {
		t.Fatal("Pool ID should not be empty")
	}

	// Read resource pool
	readName, readCpuMin, readCpuMinExp, readCpuMax, readCpuShares,
		readMemMin, readMemMinExp, readMemMax, readMemShares, err := resourcePoolRead_govmomi(config, poolID)
	if err != nil {
		t.Fatalf("Failed to read resource pool: %v", err)
	}

	// Verify properties
	if readName != poolName {
		t.Errorf("Expected name %s, got %s", poolName, readName)
	}

	if readCpuMin != cpuMin {
		t.Errorf("Expected CPU min %d, got %d", cpuMin, readCpuMin)
	}

	if readCpuMinExp != cpuMinExpandable {
		t.Errorf("Expected CPU min expandable %s, got %s", cpuMinExpandable, readCpuMinExp)
	}

	if readMemMin != memMin {
		t.Errorf("Expected mem min %d, got %d", memMin, readMemMin)
	}

	if readMemMinExp != memMinExpandable {
		t.Errorf("Expected mem min expandable %s, got %s", memMinExpandable, readMemMinExp)
	}

	t.Logf("Resource pool read: name=%s, cpu_min=%d, cpu_max=%d, cpu_shares=%s, mem_min=%d, mem_max=%d, mem_shares=%s",
		readName, readCpuMin, readCpuMax, readCpuShares, readMemMin, readMemMax, readMemShares)

	// Delete resource pool
	err = resourcePoolDelete_govmomi(config, poolID)
	if err != nil {
		t.Fatalf("Failed to delete resource pool: %v", err)
	}

	// Verify deletion - read should fail
	_, _, _, _, _, _, _, _, _, err = resourcePoolRead_govmomi(config, poolID)
	if err == nil {
		t.Error("Expected error reading deleted resource pool")
	}
}

// TestResourcePoolUpdateGovmomi tests resource pool update
func TestResourcePoolUpdateGovmomi(t *testing.T) {
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
		useGovmomi:      true,
	}

	_, err = config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Create initial pool
	poolName := "test-pool-update"
	cpuMin := 100
	cpuMinExpandable := "true"
	cpuMax := 200
	cpuShares := "normal"
	memMin := 256
	memMinExpandable := "true"
	memMax := 512
	memShares := "normal"

	poolID, err := resourcePoolCreate_govmomi(config, poolName, cpuMin, cpuMinExpandable,
		cpuMax, cpuShares, memMin, memMinExpandable, memMax, memShares, "")
	if err != nil {
		t.Fatalf("Failed to create resource pool: %v", err)
	}
	defer resourcePoolDelete_govmomi(config, poolID)

	// Update pool with new values
	newName := "test-pool-renamed"
	newCpuMin := 200
	newCpuMax := 400
	newCpuShares := "high"
	newMemMin := 512
	newMemMax := 1024
	newMemShares := "low"

	err = resourcePoolUpdate_govmomi(config, poolID, newName, newCpuMin, cpuMinExpandable,
		newCpuMax, newCpuShares, newMemMin, memMinExpandable, newMemMax, newMemShares)
	if err != nil {
		t.Fatalf("Failed to update resource pool: %v", err)
	}

	// Read and verify updates
	readName, readCpuMin, _, readCpuMax, readCpuShares,
		readMemMin, _, readMemMax, readMemShares, err := resourcePoolRead_govmomi(config, poolID)
	if err != nil {
		t.Fatalf("Failed to read updated resource pool: %v", err)
	}

	if readName != newName {
		t.Errorf("Expected name %s, got %s", newName, readName)
	}

	if readCpuMin != newCpuMin {
		t.Errorf("Expected CPU min %d, got %d", newCpuMin, readCpuMin)
	}

	if readCpuMax != newCpuMax {
		t.Errorf("Expected CPU max %d, got %d", newCpuMax, readCpuMax)
	}

	if readCpuShares != newCpuShares {
		t.Errorf("Expected CPU shares %s, got %s", newCpuShares, readCpuShares)
	}

	if readMemMin != newMemMin {
		t.Errorf("Expected mem min %d, got %d", newMemMin, readMemMin)
	}

	if readMemMax != newMemMax {
		t.Errorf("Expected mem max %d, got %d", newMemMax, readMemMax)
	}

	if readMemShares != newMemShares {
		t.Errorf("Expected mem shares %s, got %s", newMemShares, readMemShares)
	}
}

// TestGetPoolIDAndNameGovmomi tests pool ID and name lookup
func TestGetPoolIDAndNameGovmomi(t *testing.T) {
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
		useGovmomi:      true,
	}

	_, err = config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Create a test pool
	poolName := "lookup-test-pool"
	poolID, err := resourcePoolCreate_govmomi(config, poolName, 100, "true", 200, "normal",
		256, "true", 512, "normal", "")
	if err != nil {
		t.Fatalf("Failed to create resource pool: %v", err)
	}
	defer resourcePoolDelete_govmomi(config, poolID)

	// Test getPoolID_govmomi
	foundID, err := getPoolID_govmomi(config, poolName)
	if err != nil {
		t.Fatalf("Failed to get pool ID by name: %v", err)
	}

	if foundID != poolID {
		t.Errorf("Expected pool ID %s, got %s", poolID, foundID)
	}

	// Test getPoolNAME_govmomi
	foundName, err := getPoolNAME_govmomi(config, poolID)
	if err != nil {
		t.Fatalf("Failed to get pool name by ID: %v", err)
	}

	if foundName != poolName {
		t.Errorf("Expected pool name %s, got %s", poolName, foundName)
	}

	// Test with non-existent pool
	_, err = getPoolID_govmomi(config, "non-existent-pool")
	if err == nil {
		t.Error("Expected error for non-existent pool")
	}
}

// TestBuildAllocationInfoShares tests share level mapping
func TestBuildAllocationInfoShares(t *testing.T) {
	tests := []struct {
		name          string
		shares        string
		expectLevel   string
		expectCustom  bool
		customValue   int32
	}{
		{
			name:        "Low shares",
			shares:      "low",
			expectLevel: "low",
			expectCustom: false,
		},
		{
			name:        "Normal shares",
			shares:      "normal",
			expectLevel: "normal",
			expectCustom: false,
		},
		{
			name:        "High shares",
			shares:      "high",
			expectLevel: "high",
			expectCustom: false,
		},
		{
			name:         "Custom shares",
			shares:       "5000",
			expectLevel:  "custom",
			expectCustom: true,
			customValue:  5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allocation := buildAllocationInfo(100, "true", 200, tt.shares)

			if allocation.Shares == nil {
				t.Fatal("Shares should not be nil")
			}

			level := string(allocation.Shares.Level)
			if level != tt.expectLevel {
				t.Errorf("Expected level %s, got %s", tt.expectLevel, level)
			}

			if tt.expectCustom {
				if allocation.Shares.Shares != tt.customValue {
					t.Errorf("Expected custom shares %d, got %d", tt.customValue, allocation.Shares.Shares)
				}
			}
		})
	}
}
