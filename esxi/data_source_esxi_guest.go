package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceGuest() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGuestRead,

		Schema: map[string]*schema.Schema{
			"guest_name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"vmid"},
				Description:   "The name of the guest VM to look up.",
			},
			"vmid": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"guest_name"},
				Description:   "The VM ID of the guest to look up. Alternative to guest_name.",
			},
			"boot_firmware": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Boot firmware type (bios or efi).",
			},
			"disk_store": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ESXi datastore where boot disk is located.",
			},
			"resource_pool_name": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Resource pool name where guest is located.",
			},
			"boot_disk_type": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest boot disk type (thin, zeroedthick, eagerzeroedthick).",
			},
			"boot_disk_size": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest boot disk size in GB.",
			},
			"memsize": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest memory size in MB.",
			},
			"numvcpus": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest number of virtual CPUs.",
			},
			"virthwver": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest virtual hardware version.",
			},
			"guestos": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest OS type.",
			},
			"network_interfaces": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"virtual_network": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Virtual network name.",
						},
						"mac_address": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "MAC address.",
						},
						"nic_type": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "NIC type.",
						},
					},
				},
			},
			"power": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest power state.",
			},
			"ip_address": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address reported by VMware tools.",
			},
			"virtual_disks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"virtual_disk_id": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Virtual disk identifier.",
						},
						"slot": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SCSI slot (e.g., 0:1).",
						},
					},
				},
			},
			"notes": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Guest notes (annotation).",
			},
			"guestinfo": &schema.Schema{
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Guest info variables.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"pci_controllers": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "PCI controllers (SCSI, IDE, SATA, NVMe).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"network_adapters": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Network adapters with detailed device info.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"address_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"connected": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"start_connected": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"disk_drives": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Virtual disk drives.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"controller_key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"unit_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"capacity_gb": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"thin_provisioned": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"file_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"cdrom_drives": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "CD/DVD-ROM drives.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"controller_key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"unit_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"video_cards": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Video cards.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"video_ram_kb": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"usb_devices": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "USB controllers.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceGuestRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[dataSourceGuestRead]")

	// Step 1: Resolve VM ID
	var vmid string
	var err error

	guest_name_val, guest_name_ok := d.GetOk("guest_name")
	vmid_val, vmid_ok := d.GetOk("vmid")

	// Ensure at least one is provided
	if !guest_name_ok && !vmid_ok {
		return fmt.Errorf("Either 'guest_name' or 'vmid' must be provided")
	}

	if guest_name_ok {
		guest_name := guest_name_val.(string)
		// Look up by name
		vmid, err = guestGetVMID(c, guest_name)
		if err != nil {
			return fmt.Errorf("Unable to find VM with name '%s': %s", guest_name, err)
		}
	} else if vmid_ok {
		vm_id := vmid_val.(string)
		// Validate by ID
		vmid, err = guestValidateVMID(c, vm_id)
		if err != nil {
			return fmt.Errorf("Invalid VM ID '%s': %s", vm_id, err)
		}
	}

	// Step 2: Read VM data (reuse existing function)
	// Pass guest_startup_timeout=0 for data sources (don't wait for IP)
	guest_name, disk_store, disk_size, boot_disk_type, resource_pool_name,
		memsize, numvcpus, virthwver, guestos, ip_address, virtual_networks,
		boot_firmware, virtual_disks, power, notes, guestinfo, err :=
		guestREAD(c, vmid, 0)

	if err != nil || guest_name == "" {
		return fmt.Errorf("Failed to read VM with ID %s: %s", vmid, err)
	}

	// Step 3: Set ID and all computed fields
	d.SetId(vmid)
	d.Set("guest_name", guest_name)
	d.Set("disk_store", disk_store)
	d.Set("boot_disk_size", disk_size)
	if boot_disk_type != "Unknown" && boot_disk_type != "" {
		d.Set("boot_disk_type", boot_disk_type)
	}
	d.Set("resource_pool_name", resource_pool_name)
	d.Set("memsize", memsize)
	d.Set("numvcpus", numvcpus)
	d.Set("virthwver", virthwver)
	d.Set("guestos", guestos)
	d.Set("ip_address", ip_address)
	d.Set("power", power)
	d.Set("notes", notes)
	d.Set("boot_firmware", boot_firmware)
	if len(guestinfo) != 0 {
		d.Set("guestinfo", guestinfo)
	}

	// Process network interfaces (same logic as resourceGUESTRead)
	log.Printf("virtual_networks: %q\n", virtual_networks)
	nics := make([]map[string]interface{}, 0, 1)

	if virtual_networks[0][0] == "" {
		nics = nil
	}

	for nic := 0; nic < 10; nic++ {
		if virtual_networks[nic][0] != "" {
			out := make(map[string]interface{})
			out["virtual_network"] = virtual_networks[nic][0]
			out["mac_address"] = virtual_networks[nic][1]
			out["nic_type"] = virtual_networks[nic][2]
			nics = append(nics, out)
		}
	}
	d.Set("network_interfaces", nics)

	// Process virtual disks (same logic as resourceGUESTRead)
	log.Printf("virtual_disks: %q\n", virtual_disks)
	vdisks := make([]map[string]interface{}, 0, 1)

	if virtual_disks[0][0] == "" {
		vdisks = nil
	}

	for vdisk := 0; vdisk < 60; vdisk++ {
		if virtual_disks[vdisk][0] != "" {
			out := make(map[string]interface{})
			out["virtual_disk_id"] = virtual_disks[vdisk][0]
			out["slot"] = virtual_disks[vdisk][1]
			vdisks = append(vdisks, out)
		}
	}
	d.Set("virtual_disks", vdisks)

	// Read device info
	deviceInfo, err := guestReadDevices(c, vmid)
	if err != nil {
		log.Printf("[dataSourceGuestRead] Warning: failed to read device info: %s", err)
	} else {
		d.Set("pci_controllers", deviceInfo.PCIControllers)
		d.Set("network_adapters", deviceInfo.NetworkAdapters)
		d.Set("disk_drives", deviceInfo.DiskDrives)
		d.Set("cdrom_drives", deviceInfo.CDROMDrives)
		d.Set("video_cards", deviceInfo.VideoCards)
		d.Set("usb_devices", deviceInfo.USBDevices)
	}

	return nil
}
