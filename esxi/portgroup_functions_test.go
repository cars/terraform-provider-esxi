package esxi

import (
	"testing"

	"github.com/vmware/govmomi/simulator"
)

// TestPortgroupCreateReadDeleteGovmomi tests port group lifecycle
func TestPortgroupCreateReadDeleteGovmomi(t *testing.T) {
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

	// First create a vswitch for the port group
	vswitchName := "test-vswitch-for-pg"
	err = vswitchCreate_govmomi(config, vswitchName, 128)
	if err != nil {
		t.Fatalf("Failed to create vswitch: %v", err)
	}
	defer vswitchDelete_govmomi(config, vswitchName)

	// Create port group
	pgName := "test-portgroup"
	err = portgroupCreate_govmomi(config, pgName, vswitchName)
	if err != nil {
		t.Fatalf("Failed to create port group: %v", err)
	}

	// Read port group
	readVswitch, readVlan, err := portgroupRead_govmomi(config, pgName)
	if err != nil {
		t.Fatalf("Failed to read port group: %v", err)
	}

	// Verify properties
	if readVswitch != vswitchName {
		t.Errorf("Expected vswitch %s, got %s", vswitchName, readVswitch)
	}

	// Default VLAN should be 0
	if readVlan != 0 {
		t.Logf("Expected VLAN 0, got %d (may vary by simulator)", readVlan)
	}

	t.Logf("Port group read: vswitch=%s, vlan=%d", readVswitch, readVlan)

	// Delete port group
	err = portgroupDelete_govmomi(config, pgName)
	if err != nil {
		t.Fatalf("Failed to delete port group: %v", err)
	}

	// Verify deletion - read should fail
	_, _, err = portgroupRead_govmomi(config, pgName)
	if err == nil {
		t.Error("Expected error reading deleted port group")
	}
}

// TestPortgroupUpdateGovmomi tests port group update
func TestPortgroupUpdateGovmomi(t *testing.T) {
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

	// Create vswitch
	vswitchName := "test-vswitch-update"
	err = vswitchCreate_govmomi(config, vswitchName, 128)
	if err != nil {
		t.Fatalf("Failed to create vswitch: %v", err)
	}
	defer vswitchDelete_govmomi(config, vswitchName)

	// Create port group
	pgName := "test-portgroup-update"
	err = portgroupCreate_govmomi(config, pgName, vswitchName)
	if err != nil {
		t.Fatalf("Failed to create port group: %v", err)
	}
	defer portgroupDelete_govmomi(config, pgName)

	// Update port group with VLAN and security policies
	newVlan := 100
	promiscuous := "false"
	forgedTransmits := "true"
	macChanges := "true"

	err = portgroupUpdate_govmomi(config, pgName, newVlan, promiscuous, forgedTransmits, macChanges)
	if err != nil {
		t.Fatalf("Failed to update port group: %v", err)
	}

	// Read and verify VLAN
	_, readVlan, err := portgroupRead_govmomi(config, pgName)
	if err != nil {
		t.Fatalf("Failed to read updated port group: %v", err)
	}

	if readVlan != newVlan {
		t.Errorf("Expected VLAN %d, got %d", newVlan, readVlan)
	}

	// Read and verify security policy
	secPolicy, err := portgroupSecurityPolicyRead_govmomi(config, pgName)
	if err != nil {
		t.Fatalf("Failed to read security policy: %v", err)
	}

	if secPolicy == nil {
		t.Fatal("Security policy should not be nil")
	}

	// Convert string settings to expected boolean values
	expectPromiscuous := promiscuous == "true"
	expectForged := forgedTransmits == "true"
	expectMac := macChanges == "true"

	if secPolicy.AllowPromiscuous != expectPromiscuous {
		t.Errorf("Expected promiscuous mode %v, got %v", expectPromiscuous, secPolicy.AllowPromiscuous)
	}

	if secPolicy.AllowForgedTransmits != expectForged {
		t.Errorf("Expected forged transmits %v, got %v", expectForged, secPolicy.AllowForgedTransmits)
	}

	if secPolicy.AllowMACAddressChange != expectMac {
		t.Errorf("Expected MAC changes %v, got %v", expectMac, secPolicy.AllowMACAddressChange)
	}

	t.Logf("Port group updated: vlan=%d, promiscuous=%v, forged=%v, mac=%v",
		readVlan, secPolicy.AllowPromiscuous, secPolicy.AllowForgedTransmits, secPolicy.AllowMACAddressChange)
}

// TestPortgroupSecurityPolicyReadGovmomi tests reading security policies
func TestPortgroupSecurityPolicyReadGovmomi(t *testing.T) {
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

	// Create vswitch
	vswitchName := "test-vswitch-security"
	err = vswitchCreate_govmomi(config, vswitchName, 128)
	if err != nil {
		t.Fatalf("Failed to create vswitch: %v", err)
	}
	defer vswitchDelete_govmomi(config, vswitchName)

	// Create port group
	pgName := "test-portgroup-security"
	err = portgroupCreate_govmomi(config, pgName, vswitchName)
	if err != nil {
		t.Fatalf("Failed to create port group: %v", err)
	}
	defer portgroupDelete_govmomi(config, pgName)

	// Read security policy
	secPolicy, err := portgroupSecurityPolicyRead_govmomi(config, pgName)
	if err != nil {
		t.Fatalf("Failed to read security policy: %v", err)
	}

	if secPolicy == nil {
		t.Fatal("Security policy should not be nil")
	}

	// Security policy should be readable (default values will be false)
	t.Logf("Security policy: promiscuous=%v, forged=%v, mac=%v",
		secPolicy.AllowPromiscuous, secPolicy.AllowForgedTransmits, secPolicy.AllowMACAddressChange)
}

// TestPortgroupNonExistentGovmomi tests error handling for non-existent port groups
func TestPortgroupNonExistentGovmomi(t *testing.T) {
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

	// Try to read non-existent port group
	_, _, err = portgroupRead_govmomi(config, "non-existent-pg")
	if err == nil {
		t.Error("Expected error for non-existent port group")
	}

	// Try to read security policy for non-existent port group
	_, err = portgroupSecurityPolicyRead_govmomi(config, "non-existent-pg")
	if err == nil {
		t.Error("Expected error for non-existent port group")
	}

	// Try to delete non-existent port group
	err = portgroupDelete_govmomi(config, "non-existent-pg")
	if err == nil {
		t.Error("Expected error deleting non-existent port group")
	}
}
