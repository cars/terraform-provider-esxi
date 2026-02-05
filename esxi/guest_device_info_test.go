package esxi

import (
	"testing"

	"github.com/vmware/govmomi/simulator"
)

// TestGuestDeviceInfoBasic tests that device info retrieval returns expected device categories
func TestGuestDeviceInfoBasic(t *testing.T) {
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
		useGovmomi:      true,
	}

	client, err := config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	// Find a VM in the simulator
	vms, err := client.Finder.VirtualMachineList(client.Context(), "*")
	if err != nil || len(vms) == 0 {
		t.Fatal("Failed to find VMs in simulator")
	}

	vmid := vms[0].Reference().Value

	// Call device info function
	deviceInfo, err := guestReadDevices_govmomi(config, vmid)
	if err != nil {
		t.Fatalf("Failed to read device info: %v", err)
	}

	if deviceInfo == nil {
		t.Fatal("DeviceInfo should not be nil")
	}

	// Simulator VMs should have at least one controller
	if len(deviceInfo.PCIControllers) == 0 {
		t.Error("Expected at least one PCI controller")
	}

	// Log what we found for diagnostic purposes
	t.Logf("Found %d PCI controllers", len(deviceInfo.PCIControllers))
	for _, c := range deviceInfo.PCIControllers {
		t.Logf("  Controller: name=%s type=%s label=%s summary=%s key=%d",
			c["name"], c["type"], c["label"], c["summary"], c["key"])
	}

	t.Logf("Found %d network adapters", len(deviceInfo.NetworkAdapters))
	for _, n := range deviceInfo.NetworkAdapters {
		t.Logf("  NIC: name=%s type=%s mac=%s connected=%v",
			n["name"], n["type"], n["mac_address"], n["connected"])
	}

	t.Logf("Found %d disk drives", len(deviceInfo.DiskDrives))
	for _, d := range deviceInfo.DiskDrives {
		t.Logf("  Disk: name=%s capacity_gb=%d file_name=%s",
			d["name"], d["capacity_gb"], d["file_name"])
	}

	t.Logf("Found %d CD-ROM drives", len(deviceInfo.CDROMDrives))
	t.Logf("Found %d video cards", len(deviceInfo.VideoCards))
	t.Logf("Found %d USB devices", len(deviceInfo.USBDevices))
}

// TestGuestDeviceInfoControllerFields verifies controller field structure
func TestGuestDeviceInfoControllerFields(t *testing.T) {
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
		useGovmomi:      true,
	}

	_, err = config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	client := config.govmomiClient
	vms, err := client.Finder.VirtualMachineList(client.Context(), "*")
	if err != nil || len(vms) == 0 {
		t.Fatal("Failed to find VMs in simulator")
	}

	vmid := vms[0].Reference().Value

	deviceInfo, err := guestReadDevices_govmomi(config, vmid)
	if err != nil {
		t.Fatalf("Failed to read device info: %v", err)
	}

	// Verify controller fields are populated
	for i, ctrl := range deviceInfo.PCIControllers {
		if ctrl["name"] == nil || ctrl["name"] == "" {
			t.Errorf("Controller %d: name should not be empty", i)
		}
		if ctrl["type"] == nil || ctrl["type"] == "" {
			t.Errorf("Controller %d: type should not be empty", i)
		}
		validTypes := map[string]bool{"scsi": true, "ide": true, "sata": true, "nvme": true}
		if ctrlType, ok := ctrl["type"].(string); ok {
			if !validTypes[ctrlType] {
				t.Errorf("Controller %d: unexpected type %q", i, ctrlType)
			}
		}
		if ctrl["key"] == nil {
			t.Errorf("Controller %d: key should not be nil", i)
		}
	}
}

// TestGuestDeviceInfoDiskFields verifies disk drive field structure
func TestGuestDeviceInfoDiskFields(t *testing.T) {
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
		useGovmomi:      true,
	}

	_, err = config.GetGovmomiClient()
	if err != nil {
		t.Fatalf("Failed to create govmomi client: %v", err)
	}
	defer config.CloseGovmomiClient()

	client := config.govmomiClient
	vms, err := client.Finder.VirtualMachineList(client.Context(), "*")
	if err != nil || len(vms) == 0 {
		t.Fatal("Failed to find VMs in simulator")
	}

	vmid := vms[0].Reference().Value

	deviceInfo, err := guestReadDevices_govmomi(config, vmid)
	if err != nil {
		t.Fatalf("Failed to read device info: %v", err)
	}

	// Verify disk drive fields if any exist
	for i, disk := range deviceInfo.DiskDrives {
		if disk["name"] == nil || disk["name"] == "" {
			t.Errorf("Disk %d: name should not be empty", i)
		}
		if disk["key"] == nil {
			t.Errorf("Disk %d: key should not be nil", i)
		}
		if disk["controller_key"] == nil {
			t.Errorf("Disk %d: controller_key should not be nil", i)
		}
		if _, ok := disk["capacity_gb"]; !ok {
			t.Errorf("Disk %d: capacity_gb should be present", i)
		}
		if _, ok := disk["thin_provisioned"]; !ok {
			t.Errorf("Disk %d: thin_provisioned should be present", i)
		}
	}
}
