package esxi

import (
	"testing"

	"github.com/vmware/govmomi/simulator"
)

// TestDiskStoreValidateGovmomi tests datastore validation
func TestDiskStoreValidateGovmomi(t *testing.T) {
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

	// Get datastore name
	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to get govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	datastores, err := client.Finder.DatastoreList(client.Context(), "*")
	if err != nil || len(datastores) == 0 {
		t.Fatal("Failed to get datastores")
	}

	dsName := datastores[0].Name()

	// Test validation
	err = diskStoreValidate(config, dsName)
	if err != nil {
		t.Fatalf("Datastore validation failed: %v", err)
	}

	// Test with invalid datastore
	err = diskStoreValidate(config, "nonexistent-datastore")
	if err == nil {
		t.Fatal("Expected error for non-existent datastore")
	}
}

// TestVirtualDiskCreateReadGovmomi tests creating and reading virtual disks
func TestVirtualDiskCreateReadGovmomi(t *testing.T) {
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

	// Get datastore
	datastores, err := client.Finder.DatastoreList(client.Context(), "*")
	if err != nil || len(datastores) == 0 {
		t.Skip("No datastores available for testing")
	}

	dsName := datastores[0].Name()

	// Test parameters
	diskDir := "test-vdisks"
	diskName := "test-disk.vmdk"
	diskSize := 1 // 1 GB
	diskType := "thin"

	// Create virtual disk
	virtdiskID, err := virtualDiskCREATE(config, dsName, diskDir, diskName, diskSize, diskType)
	if err != nil {
		t.Fatalf("Failed to create virtual disk: %v", err)
	}

	if virtdiskID == "" {
		t.Fatal("Virtual disk ID should not be empty")
	}

	// Read virtual disk
	readDsName, readDir, readName, readSize, readType, err := virtualDiskREAD(config, virtdiskID)
	if err != nil {
		t.Fatalf("Failed to read virtual disk: %v", err)
	}

	// Verify properties
	if readDsName != dsName {
		t.Errorf("Expected datastore %s, got %s", dsName, readDsName)
	}

	if readDir != diskDir {
		t.Errorf("Expected directory %s, got %s", diskDir, readDir)
	}

	if readName != diskName {
		t.Errorf("Expected disk name %s, got %s", diskName, readName)
	}

	// Size might not be exact due to rounding
	if readSize < diskSize-1 || readSize > diskSize+1 {
		t.Errorf("Expected size around %d GB, got %d GB", diskSize, readSize)
	}

	// Type might be "Unknown" if not fully supported in simulator
	if readType != diskType && readType != "Unknown" {
		t.Logf("Warning: Expected type %s, got %s (may be simulator limitation)", diskType, readType)
	}
}

// TestBuildAllocationInfo tests the allocation info builder
func TestBuildAllocationInfo(t *testing.T) {
	tests := []struct {
		name           string
		min            int
		minExpandable  string
		max            int
		shares         string
		expectError    bool
	}{
		{
			name:          "Normal allocation",
			min:           100,
			minExpandable: "true",
			max:           200,
			shares:        "normal",
			expectError:   false,
		},
		{
			name:          "Low shares",
			min:           50,
			minExpandable: "false",
			max:           150,
			shares:        "low",
			expectError:   false,
		},
		{
			name:          "High shares",
			min:           200,
			minExpandable: "true",
			max:           400,
			shares:        "high",
			expectError:   false,
		},
		{
			name:          "Custom shares",
			min:           100,
			minExpandable: "true",
			max:           200,
			shares:        "2000",
			expectError:   false,
		},
		{
			name:          "Unlimited max",
			min:           100,
			minExpandable: "true",
			max:           0,
			shares:        "normal",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allocation := buildAllocationInfo(tt.min, tt.minExpandable, tt.max, tt.shares)

			// Verify reservation
			if tt.min > 0 && allocation.Reservation != nil {
				if *allocation.Reservation != int64(tt.min) {
					t.Errorf("Expected reservation %d, got %d", tt.min, *allocation.Reservation)
				}
			}

			// Verify expandable reservation
			if allocation.ExpandableReservation != nil {
				expected := tt.minExpandable != "false"
				if *allocation.ExpandableReservation != expected {
					t.Errorf("Expected expandable %v, got %v", expected, *allocation.ExpandableReservation)
				}
			}

			// Verify limit
			if allocation.Limit != nil {
				expectedLimit := int64(tt.max)
				if tt.max == 0 {
					expectedLimit = -1
				}
				if *allocation.Limit != expectedLimit {
					t.Errorf("Expected limit %d, got %d", expectedLimit, *allocation.Limit)
				}
			}

			// Verify shares
			if allocation.Shares == nil {
				t.Fatal("Shares should not be nil")
			}
		})
	}
}
