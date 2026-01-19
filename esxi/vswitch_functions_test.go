package esxi

import (
	"testing"

	"github.com/vmware/govmomi/simulator"
)

// TestVswitchCreateReadDeleteGovmomi tests vswitch lifecycle
func TestVswitchCreateReadDeleteGovmomi(t *testing.T) {
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

	vswitchName := "test-vswitch"
	ports := 128

	// Create vswitch
	err = vswitchCreate_govmomi(config, vswitchName, ports)
	if err != nil {
		t.Fatalf("Failed to create vswitch: %v", err)
	}

	// Read vswitch
	readPorts, readMtu, readUplinks, readLinkDiscovery, readPromiscuous, readMacChanges, readForgedTransmits, err := vswitchRead_govmomi(config, vswitchName)
	if err != nil {
		t.Fatalf("Failed to read vswitch: %v", err)
	}

	// Verify properties
	if readPorts != ports {
		t.Errorf("Expected %d ports, got %d", ports, readPorts)
	}

	// MTU should have a default value
	if readMtu <= 0 {
		t.Errorf("MTU should be positive, got %d", readMtu)
	}

	// Uplinks should be initialized (possibly empty)
	if readUplinks == nil {
		t.Error("Uplinks should not be nil")
	}

	// Link discovery should have a default
	if readLinkDiscovery == "" {
		t.Error("Link discovery mode should not be empty")
	}

	t.Logf("VSwitch read: ports=%d, mtu=%d, uplinks=%v, linkDiscovery=%s, promiscuous=%v, macChanges=%v, forgedTransmits=%v",
		readPorts, readMtu, readUplinks, readLinkDiscovery, readPromiscuous, readMacChanges, readForgedTransmits)

	// Delete vswitch
	err = vswitchDelete_govmomi(config, vswitchName)
	if err != nil {
		t.Fatalf("Failed to delete vswitch: %v", err)
	}

	// Verify deletion - read should fail
	_, _, _, _, _, _, _, err = vswitchRead_govmomi(config, vswitchName)
	if err == nil {
		t.Error("Expected error reading deleted vswitch")
	}
}

// TestVswitchUpdateGovmomi tests vswitch update
func TestVswitchUpdateGovmomi(t *testing.T) {
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

	vswitchName := "test-vswitch-update"
	ports := 128

	// Create vswitch
	err = vswitchCreate_govmomi(config, vswitchName, ports)
	if err != nil {
		t.Fatalf("Failed to create vswitch: %v", err)
	}
	defer vswitchDelete_govmomi(config, vswitchName)

	// Update vswitch
	newMtu := 9000
	uplinks := []string{}
	linkDiscoveryMode := "listen"
	promiscuousMode := false
	macChanges := true
	forgedTransmits := true

	err = vswitchUpdate_govmomi(config, vswitchName, ports, newMtu, uplinks,
		linkDiscoveryMode, promiscuousMode, macChanges, forgedTransmits)
	if err != nil {
		t.Fatalf("Failed to update vswitch: %v", err)
	}

	// Read and verify
	_, readMtu, _, readLinkDiscovery, readPromiscuous, readMacChanges, readForgedTransmits, err := vswitchRead_govmomi(config, vswitchName)
	if err != nil {
		t.Fatalf("Failed to read updated vswitch: %v", err)
	}

	if readMtu != newMtu {
		t.Errorf("Expected MTU %d, got %d", newMtu, readMtu)
	}

	if readLinkDiscovery != linkDiscoveryMode {
		t.Errorf("Expected link discovery %s, got %s", linkDiscoveryMode, readLinkDiscovery)
	}

	if readPromiscuous != promiscuousMode {
		t.Errorf("Expected promiscuous mode %v, got %v", promiscuousMode, readPromiscuous)
	}

	if readMacChanges != macChanges {
		t.Errorf("Expected MAC changes %v, got %v", macChanges, readMacChanges)
	}

	if readForgedTransmits != forgedTransmits {
		t.Errorf("Expected forged transmits %v, got %v", forgedTransmits, readForgedTransmits)
	}
}

// TestInArrayOfStrings tests the utility function
func TestInArrayOfStrings(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		val      string
		expected bool
	}{
		{
			name:     "Found in array",
			slice:    []string{"a", "b", "c"},
			val:      "b",
			expected: true,
		},
		{
			name:     "Not found in array",
			slice:    []string{"a", "b", "c"},
			val:      "d",
			expected: false,
		},
		{
			name:     "Empty array",
			slice:    []string{},
			val:      "a",
			expected: false,
		},
		{
			name:     "Empty string search",
			slice:    []string{"a", "", "c"},
			val:      "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inArrayOfStrings(tt.slice, tt.val)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
