package esxi

import (
	"fmt"
	"log"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// DeviceInfo holds all categorized device information for a VM
type DeviceInfo struct {
	PCIControllers  []map[string]interface{}
	NetworkAdapters []map[string]interface{}
	DiskDrives      []map[string]interface{}
	CDROMDrives     []map[string]interface{}
	VideoCards      []map[string]interface{}
	USBDevices      []map[string]interface{}
}

// guestReadDevices retrieves device info for a VM using govmomi
func guestReadDevices(c *Config, vmid string) (*DeviceInfo, error) {
	log.Println("[guestReadDevices]")

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get govmomi client: %w", err)
	}

	vm, err := getVMByID(gc, vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to find VM: %w", err)
	}

	var mvm mo.VirtualMachine
	err = vm.Properties(gc.Context(), vm.Reference(), []string{"config.hardware.device"}, &mvm)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM device properties: %w", err)
	}

	if mvm.Config == nil {
		return nil, fmt.Errorf("VM config is nil")
	}

	info := &DeviceInfo{}

	for _, device := range mvm.Config.Hardware.Device {
		base := device.GetVirtualDevice()

		switch d := device.(type) {
		// SCSI controllers
		case *types.VirtualLsiLogicController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "scsi", "LSI Logic Parallel"))
		case *types.VirtualLsiLogicSASController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "scsi", "LSI Logic SAS"))
		case *types.VirtualBusLogicController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "scsi", "BusLogic"))
		case *types.ParaVirtualSCSIController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "scsi", "VMware Paravirtual"))

		// IDE controllers
		case *types.VirtualIDEController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "ide", "IDE"))

		// SATA controllers
		case *types.VirtualAHCIController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "sata", "AHCI"))

		// NVMe controllers
		case *types.VirtualNVMEController:
			info.PCIControllers = append(info.PCIControllers, controllerMap(base, "nvme", "NVMe"))

		// Network adapters
		case types.BaseVirtualEthernetCard:
			info.NetworkAdapters = append(info.NetworkAdapters, networkAdapterMap(d))

		// Virtual disks
		case *types.VirtualDisk:
			info.DiskDrives = append(info.DiskDrives, diskDriveMap(d))

		// CD-ROM drives
		case *types.VirtualCdrom:
			info.CDROMDrives = append(info.CDROMDrives, cdromMap(base))

		// Video cards
		case *types.VirtualMachineVideoCard:
			info.VideoCards = append(info.VideoCards, videoCardMap(base, d))

		// USB controllers
		case *types.VirtualUSBController:
			info.USBDevices = append(info.USBDevices, usbMap(base))
		case *types.VirtualUSBXHCIController:
			info.USBDevices = append(info.USBDevices, usbMap(base))
		}
	}

	return info, nil
}

func controllerMap(base *types.VirtualDevice, controllerType, summary string) map[string]interface{} {
	return map[string]interface{}{
		"name":    deviceName(base),
		"type":    controllerType,
		"label":   deviceLabel(base),
		"summary": summary,
		"key":     int(base.Key),
	}
}

func networkAdapterMap(nic types.BaseVirtualEthernetCard) map[string]interface{} {
	ethCard := nic.GetVirtualEthernetCard()
	base := ethCard.GetVirtualDevice()

	nicType := "unknown"
	switch nic.(type) {
	case *types.VirtualE1000:
		nicType = "e1000"
	case *types.VirtualE1000e:
		nicType = "e1000e"
	case *types.VirtualVmxnet3:
		nicType = "vmxnet3"
	case *types.VirtualVmxnet2:
		nicType = "vmxnet2"
	case *types.VirtualPCNet32:
		nicType = "pcnet32"
	}

	networkName := ""
	if backing, ok := ethCard.Backing.(*types.VirtualEthernetCardNetworkBackingInfo); ok {
		networkName = backing.DeviceName
	} else if backing, ok := ethCard.Backing.(*types.VirtualEthernetCardDistributedVirtualPortBackingInfo); ok {
		networkName = backing.Port.PortgroupKey
	}

	connected := false
	startConnected := false
	if ethCard.Connectable != nil {
		connected = ethCard.Connectable.Connected
		startConnected = ethCard.Connectable.StartConnected
	}

	return map[string]interface{}{
		"name":            deviceName(base),
		"type":            nicType,
		"label":           deviceLabel(base),
		"summary":         networkName,
		"mac_address":     ethCard.MacAddress,
		"address_type":    ethCard.AddressType,
		"key":             int(base.Key),
		"connected":       connected,
		"start_connected": startConnected,
	}
}

func diskDriveMap(disk *types.VirtualDisk) map[string]interface{} {
	base := disk.GetVirtualDevice()

	unitNumber := 0
	if disk.UnitNumber != nil {
		unitNumber = int(*disk.UnitNumber)
	}

	capacityGB := int(disk.CapacityInBytes / (1024 * 1024 * 1024))

	fileName := ""
	thinProvisioned := false
	if backing, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
		fileName = backing.FileName
		if backing.ThinProvisioned != nil {
			thinProvisioned = *backing.ThinProvisioned
		}
	}

	return map[string]interface{}{
		"name":             deviceName(base),
		"label":            deviceLabel(base),
		"summary":          fmt.Sprintf("%d GB", capacityGB),
		"key":              int(base.Key),
		"controller_key":   int(base.ControllerKey),
		"unit_number":      unitNumber,
		"capacity_gb":      capacityGB,
		"thin_provisioned": thinProvisioned,
		"file_name":        fileName,
	}
}

func cdromMap(base *types.VirtualDevice) map[string]interface{} {
	unitNumber := 0
	if base.UnitNumber != nil {
		unitNumber = int(*base.UnitNumber)
	}

	return map[string]interface{}{
		"name":           deviceName(base),
		"label":          deviceLabel(base),
		"summary":        deviceSummary(base),
		"key":            int(base.Key),
		"controller_key": int(base.ControllerKey),
		"unit_number":    unitNumber,
	}
}

func videoCardMap(base *types.VirtualDevice, card *types.VirtualMachineVideoCard) map[string]interface{} {
	return map[string]interface{}{
		"name":         deviceName(base),
		"label":        deviceLabel(base),
		"summary":      deviceSummary(base),
		"key":          int(base.Key),
		"video_ram_kb": int(card.VideoRamSizeInKB),
	}
}

func usbMap(base *types.VirtualDevice) map[string]interface{} {
	return map[string]interface{}{
		"name":    deviceName(base),
		"label":   deviceLabel(base),
		"summary": deviceSummary(base),
		"key":     int(base.Key),
	}
}

func deviceName(base *types.VirtualDevice) string {
	if base.DeviceInfo != nil {
		if desc, ok := base.DeviceInfo.(*types.Description); ok {
			return desc.Label
		}
	}
	return fmt.Sprintf("device-%d", base.Key)
}

func deviceLabel(base *types.VirtualDevice) string {
	if base.DeviceInfo != nil {
		if desc, ok := base.DeviceInfo.(*types.Description); ok {
			return desc.Label
		}
	}
	return ""
}

func deviceSummary(base *types.VirtualDevice) string {
	if base.DeviceInfo != nil {
		if desc, ok := base.DeviceInfo.(*types.Description); ok {
			return desc.Summary
		}
	}
	return ""
}
