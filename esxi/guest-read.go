package esxi

import (
	"bufio"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceGUESTRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceGUESTRead]")

	guest_startup_timeout := d.Get("guest_startup_timeout").(int)

	var power string

	guest_name, disk_store, disk_size, boot_disk_type, resource_pool_name, memsize, numvcpus, virthwver, guestos, ip_address, virtual_networks, boot_firmware, virtual_disks, power, notes, guestinfo, err := guestREAD(c, d.Id(), guest_startup_timeout)
	if err != nil || guest_name == "" {
		d.SetId("")
		return nil
	}

	d.Set("guest_name", guest_name)
	d.Set("disk_store", disk_store)
	d.Set("disk_size", disk_size)
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

	// Do network interfaces
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

	// Do virtual disks
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

	return nil
}

func guestREAD(c *Config, vmid string, guest_startup_timeout int) (string, string, string, string, string, string, string, string, string, string, [10][3]string, string, [60][2]string, string, string, map[string]interface{}, error) {
	esxiConnInfo := getConnectionInfo(c)
	log.Println("[guestREAD]")

	var guest_name, disk_store, virtual_disk_type, resource_pool_name, guestos, ip_address, notes string
	var dst_vmx_ds, dst_vmx, dst_vmx_file, vmx_contents, power string
	var disk_size, vdiskindex int
	var memsize, numvcpus, virthwver string
	var virtual_networks [10][3]string
	var boot_firmware string = "bios"
	var virtual_disks [60][2]string
	var guestinfo map[string]interface{}

	r, _ := regexp.Compile("")

	remote_cmd := fmt.Sprintf("vim-cmd  vmsvc/get.summary %s", vmid)
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get Guest summary")

	if strings.Contains(stdout, "Unable to find a VM corresponding") {
		return "", "", "", "", "", "", "", "", "", "", virtual_networks, "", virtual_disks, "", "", nil, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		switch {
		case strings.Contains(scanner.Text(), "name = "):
			r, _ = regexp.Compile(`\".*\"`)
			guest_name = r.FindString(scanner.Text())
			nr := strings.NewReplacer(`"`, "", `"`, "")
			guest_name = nr.Replace(guest_name)
		case strings.Contains(scanner.Text(), "vmPathName = "):
			r, _ = regexp.Compile(`\[.*\]`)
			disk_store = r.FindString(scanner.Text())
			nr := strings.NewReplacer("[", "", "]", "")
			disk_store = nr.Replace(disk_store)
		}
	}

	//  Get resource pool that this VM is located
	remote_cmd = fmt.Sprintf(`grep -A2 'objID>%s</objID' /etc/vmware/hostd/pools.xml | grep -o resourcePool.*resourcePool`, vmid)
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "check if guest is in resource pool")
	nr := strings.NewReplacer("resourcePool>", "", "</resourcePool", "")
	vm_resource_pool_id := nr.Replace(stdout)
	log.Printf("[GuestRead] resource_pool_name|%s| scanner.Text():|%s|\n", vm_resource_pool_id, stdout)
	resource_pool_name, err = getPoolNAME(c, vm_resource_pool_id)
	log.Printf("[GuestRead] resource_pool_name|%s| scanner.Text():|%s|\n", vm_resource_pool_id, err)

	//
	//  Read vmx file into memory to read settings
	//
	//      -Get location of vmx file on esxi host
	remote_cmd = fmt.Sprintf("vim-cmd vmsvc/get.config %s | grep vmPathName|grep -oE \"\\[.*\\]\"", vmid)
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get dst_vmx_ds")
	dst_vmx_ds = stdout
	dst_vmx_ds = strings.Trim(dst_vmx_ds, "[")
	dst_vmx_ds = strings.Trim(dst_vmx_ds, "]")

	remote_cmd = fmt.Sprintf("vim-cmd vmsvc/get.config %s | grep vmPathName|awk '{print $NF}'|sed 's/[\"|,]//g'", vmid)
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get dst_vmx")
	dst_vmx = stdout

	dst_vmx_file = "/vmfs/volumes/" + dst_vmx_ds + "/" + dst_vmx

	log.Printf("[guestREAD] dst_vmx_file: %s\n", dst_vmx_file)
	log.Printf("[guestREAD] disk_store: %s  dst_vmx_ds:%s\n", disk_store, dst_vmx_file)

	remote_cmd = fmt.Sprintf("cat \"%s\"", dst_vmx_file)
	vmx_contents, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "read guest_name.vmx file")

	// Used to keep track if a network interface is using static or generated macs.
	var isGeneratedMAC [10]bool

	//  Read vmx_contents line-by-line to get current settings.
	vdiskindex = 0
	scanner = bufio.NewScanner(strings.NewReader(vmx_contents))
	for scanner.Scan() {

		switch {
		case strings.Contains(scanner.Text(), "memSize = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			nr = strings.NewReplacer(`"`, "", `"`, "")
			memsize = nr.Replace(stdout)
			log.Printf("[guestREAD] memsize found: %s\n", memsize)

		case strings.Contains(scanner.Text(), "numvcpus = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			nr = strings.NewReplacer(`"`, "", `"`, "")
			numvcpus = nr.Replace(stdout)
			log.Printf("[guestREAD] numvcpus found: %s\n", numvcpus)

		case strings.Contains(scanner.Text(), "numa.autosize.vcpu."):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			nr = strings.NewReplacer(`"`, "", `"`, "")
			numvcpus = nr.Replace(stdout)
			log.Printf("[guestREAD] numa.vcpu (numvcpus) found: %s\n", numvcpus)

		case strings.Contains(scanner.Text(), "virtualHW.version = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			virthwver = strings.Replace(stdout, `"`, "", -1)
			log.Printf("[guestREAD] virthwver found: %s\n", virthwver)

		case strings.Contains(scanner.Text(), "guestOS = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			guestos = strings.Replace(stdout, `"`, "", -1)
			log.Printf("[guestREAD] guestos found: %s\n", guestos)

		case strings.Contains(scanner.Text(), "scsi"):
			re := regexp.MustCompile("scsi([0-3]):([0-9]{1,2}).(.*) = \"(.*)\"")
			results := re.FindStringSubmatch(scanner.Text())
			if len(results) > 4 {
				log.Printf("[guestREAD] %s : %s . %s = %s\n", results[1], results[2], results[3], results[4])

				if (results[1] == "0") && (results[2] == "0") {
					// Skip boot disk
				} else {
					if strings.Contains(results[3], "fileName") == true {
						log.Printf("[guestREAD] %s : %s\n", results[0], results[4])
						virtual_disks[vdiskindex][0] = results[4]
						virtual_disks[vdiskindex][1] = fmt.Sprintf("%s:%s", results[1], results[2])
						vdiskindex += 1
					}
				}
			}

		case strings.Contains(scanner.Text(), "ethernet"):
			re := regexp.MustCompile("ethernet(.).(.*) = \"(.*)\"")
			results := re.FindStringSubmatch(scanner.Text())
			index, _ := strconv.Atoi(results[1])

			switch results[2] {
			case "networkName":
				virtual_networks[index][0] = results[3]
				log.Printf("[guestREAD] %s : %s\n", results[0], results[3])

			case "addressType":
				if results[3] == "generated" {
					isGeneratedMAC[index] = true
				}

				//  Done't save generatedAddress...   It should not be saved because it
			//  should be considered dynamic & is breaks the update MAC address code.
			//case "generatedAddress":
			//	if isGeneratedMAC[index] == true {
			//		virtual_networks[index][1] = results[3]
			//		log.Printf("[guestREAD] %s : %s\n", results[0], results[3])
			//	}

			case "address":
				if isGeneratedMAC[index] == false {
					virtual_networks[index][1] = results[3]
					log.Printf("[guestREAD] %s : %s\n", results[0], results[3])
				}

			case "virtualDev":
				virtual_networks[index][2] = results[3]
				log.Printf("[guestREAD] %s : %s\n", results[0], results[3])
			}

		case strings.Contains(scanner.Text(), "firmware = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			boot_firmware = strings.Replace(stdout, `"`, "", -1)
			log.Printf("[guestREAD] firmware found: %s\n", boot_firmware)

		case strings.Contains(scanner.Text(), "annotation = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			notes = strings.Replace(stdout, `"`, "", -1)
			notes = strings.Replace(notes, "|22", "\"", -1)
			log.Printf("[guestREAD] annotation found: %s\n", notes)

		}
	}

	parsed_vmx := ParseVMX(vmx_contents)

	//  Get power state
	log.Println("guestREAD: guestPowerGetState")
	power = guestPowerGetState(c, vmid)

	//
	// Get IP address (need vmware tools installed)
	//
	if power == "on" {
		ip_address = guestGetIpAddress(c, vmid, guest_startup_timeout)
		log.Printf("[guestREAD] guestGetIpAddress: %s\n", ip_address)
	} else {
		ip_address = ""
	}

	// Get boot disk size
	boot_disk_vmdkPATH, _ := getBootDiskPath(c, vmid)
	_, _, _, disk_size, virtual_disk_type, err = virtualDiskREAD(c, boot_disk_vmdkPATH)
	str_disk_size := strconv.Itoa(disk_size)

	// Get guestinfo value
	guestinfo = make(map[string]interface{})
	for key, value := range parsed_vmx {
		if strings.Contains(key, "guestinfo") {
			short_key := strings.Replace(key, "guestinfo.", "", -1)
			guestinfo[short_key] = value
		}
	}

	// return results
	return guest_name, disk_store, str_disk_size, virtual_disk_type, resource_pool_name, memsize, numvcpus, virthwver, guestos, ip_address, virtual_networks, boot_firmware, virtual_disks, power, notes, guestinfo, err
}

// guestREAD_govmomi reads VM properties using govmomi
func guestREAD_govmomi(c *Config, vmid string, guest_startup_timeout int) (string, string, string, string, string, string, string, string, string, string, [10][3]string, string, [60][2]string, string, string, map[string]interface{}, error) {
	log.Println("[guestREAD_govmomi]")

	var guest_name, disk_store, virtual_disk_type, resource_pool_name, guestos, ip_address, notes string
	var disk_size int
	var memsize, numvcpus, virthwver string
	var virtual_networks [10][3]string
	var boot_firmware string = "bios"
	var virtual_disks [60][2]string
	var guestinfo map[string]interface{}
	var power string

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", virtual_networks, "", virtual_disks, "", "", nil, err
	}

	vm, err := getVMByID(gc, vmid)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", virtual_networks, "", virtual_disks, "", "", nil, err
	}

	// Get VM properties
	var mvm mo.VirtualMachine
	err = vm.Properties(gc.Context(), vm.Reference(), []string{
		"name",
		"config",
		"runtime",
		"guest",
		"resourcePool",
	}, &mvm)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", virtual_networks, "", virtual_disks, "", "", nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Basic properties
	guest_name = mvm.Name
	memsize = fmt.Sprintf("%d", mvm.Config.Hardware.MemoryMB)
	numvcpus = fmt.Sprintf("%d", mvm.Config.Hardware.NumCPU)
	virthwver = strings.TrimPrefix(mvm.Config.Version, "vmx-")
	guestos = mvm.Config.GuestId
	notes = mvm.Config.Annotation

	// Get datastore name from VM files
	if len(mvm.Config.Files.VmPathName) > 0 {
		// VmPathName format: "[datastore] path/to/vm.vmx"
		vmPathName := mvm.Config.Files.VmPathName
		if idx := strings.Index(vmPathName, "["); idx >= 0 {
			if endIdx := strings.Index(vmPathName, "]"); endIdx > idx {
				disk_store = vmPathName[idx+1 : endIdx]
			}
		}
	}

	// Boot firmware
	if mvm.Config.Firmware != "" {
		boot_firmware = strings.ToLower(mvm.Config.Firmware)
	}

	// Get network interfaces
	nicIndex := 0
	for _, device := range mvm.Config.Hardware.Device {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			if nicIndex >= 10 {
				break
			}

			ethCard := nic.GetVirtualEthernetCard()

			// Get network name
			if backing, ok := ethCard.Backing.(*types.VirtualEthernetCardNetworkBackingInfo); ok {
				if backing.DeviceName != "" {
					virtual_networks[nicIndex][0] = backing.DeviceName
				}
			} else if backing, ok := ethCard.Backing.(*types.VirtualEthernetCardDistributedVirtualPortBackingInfo); ok {
				// DVS port group
				if backing.Port.PortgroupKey != "" {
					virtual_networks[nicIndex][0] = backing.Port.PortgroupKey
				}
			}

			// Get MAC address (only if statically assigned)
			if ethCard.AddressType == string(types.VirtualEthernetCardMacTypeManual) ||
			   ethCard.AddressType == string(types.VirtualEthernetCardMacTypeAssigned) {
				virtual_networks[nicIndex][1] = ethCard.MacAddress
			}

			// Get NIC type
			switch nic.(type) {
			case *types.VirtualE1000:
				virtual_networks[nicIndex][2] = "e1000"
			case *types.VirtualE1000e:
				virtual_networks[nicIndex][2] = "e1000e"
			case *types.VirtualVmxnet3:
				virtual_networks[nicIndex][2] = "vmxnet3"
			case *types.VirtualVmxnet2:
				virtual_networks[nicIndex][2] = "vmxnet2"
			case *types.VirtualPCNet32:
				virtual_networks[nicIndex][2] = "pcnet32"
			default:
				virtual_networks[nicIndex][2] = "unknown"
			}

			nicIndex++
		}
	}

	// Get virtual disks (excluding boot disk)
	diskIndex := 0
	for _, device := range mvm.Config.Hardware.Device {
		if disk, ok := device.(*types.VirtualDisk); ok {
			// Skip boot disk (scsi0:0)
			if disk.UnitNumber != nil && *disk.UnitNumber == 0 {
				if ctlr := disk.ControllerKey; ctlr == 1000 { // SCSI controller 0
					continue
				}
			}

			if diskIndex >= 60 {
				break
			}

			// Get disk backing filename
			if backing, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
				virtual_disks[diskIndex][0] = backing.FileName

				// Get slot (controller:unit)
				if disk.UnitNumber != nil {
					controllerNum := (disk.ControllerKey - 1000) / 1000
					virtual_disks[diskIndex][1] = fmt.Sprintf("%d:%d", controllerNum, *disk.UnitNumber)
				}
			}

			diskIndex++
		}
	}

	// Get boot disk info
	for _, device := range mvm.Config.Hardware.Device {
		if disk, ok := device.(*types.VirtualDisk); ok {
			// Find boot disk (scsi0:0)
			if disk.UnitNumber != nil && *disk.UnitNumber == 0 {
				if ctlr := disk.ControllerKey; ctlr == 1000 { // SCSI controller 0
					if backing, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
						// Convert capacity from bytes to GB
						disk_size = int(disk.CapacityInBytes / (1024 * 1024 * 1024))

						// Determine disk type
						if backing.ThinProvisioned != nil && *backing.ThinProvisioned {
							virtual_disk_type = "thin"
						} else if backing.EagerlyScrub != nil && *backing.EagerlyScrub {
							virtual_disk_type = "eagerzeroedthick"
						} else {
							virtual_disk_type = "zeroedthick"
						}
					}
					break
				}
			}
		}
	}

	// Get resource pool name
	if mvm.ResourcePool != nil {
		rpRef := *mvm.ResourcePool
		rp := mo.ResourcePool{}
		err = gc.Client.RetrieveOne(gc.Context(), rpRef, []string{"name", "parent"}, &rp)
		if err == nil {
			resource_pool_name = rp.Name
			// If it's the root resource pool, use empty string
			if resource_pool_name == "Resources" {
				resource_pool_name = ""
			}
		}
	}

	// Get guestinfo extraConfig values
	guestinfo = make(map[string]interface{})
	for _, opt := range mvm.Config.ExtraConfig {
		if optVal, ok := opt.(*types.OptionValue); ok {
			key := optVal.Key
			if strings.HasPrefix(key, "guestinfo.") {
				shortKey := strings.TrimPrefix(key, "guestinfo.")
				if val, ok := optVal.Value.(string); ok {
					guestinfo[shortKey] = val
				}
			}
		}
	}

	// Get power state
	power = guestPowerGetState_govmomi(c, vmid)

	// Get IP address if powered on
	if power == "on" {
		ip_address = guestGetIpAddress_govmomi(c, vmid, guest_startup_timeout)
		log.Printf("[guestREAD_govmomi] guestGetIpAddress: %s\n", ip_address)
	} else {
		ip_address = ""
	}

	str_disk_size := strconv.Itoa(disk_size)

	// Return results
	return guest_name, disk_store, str_disk_size, virtual_disk_type, resource_pool_name, memsize, numvcpus, virthwver, guestos, ip_address, virtual_networks, boot_firmware, virtual_disks, power, notes, guestinfo, err
}
