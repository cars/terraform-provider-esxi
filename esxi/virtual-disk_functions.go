package esxi

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

//
//  Validate Disk Store
//
func diskStoreValidate(c *Config, disk_store string) error {
	// Use govmomi if enabled
	if c.useGovmomi {
		return diskStoreValidate_govmomi(c, disk_store)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Printf("[diskStoreValidate]\n")

	var remote_cmd, stdout string
	var err error

	//
	//  Check if Disk Store already exists
	//
	remote_cmd = fmt.Sprintf("esxcli storage filesystem list | grep '/vmfs/volumes/.*[VMFS|NFS]' |awk '{for(i=2;i<=NF-5;++i)printf $i\" \" ; printf \"\\n\"}'")
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get list of disk stores")
	if err != nil {
		return fmt.Errorf("Unable to get list of disk stores: %s\n", err)
	}
	log.Printf("1: Available Disk Stores: %s\n", strings.Replace(stdout, "\n", " ", -1))

	if strings.Contains(stdout, disk_store) == false {
		remote_cmd = fmt.Sprintf("esxcli storage filesystem rescan")
		_, _ = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Refresh filesystems")

		remote_cmd = fmt.Sprintf("esxcli storage filesystem list | grep '/vmfs/volumes/.*[VMFS|NFS]' |awk '{for(i=2;i<=NF-5;++i)printf $i\" \" ; printf \"\\n\"}'")
		stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get list of disk stores")
		if err != nil {
			return fmt.Errorf("Unable to get list of disk stores: %s\n", err)
		}
		log.Printf("2: Available Disk Stores: %s\n", strings.Replace(stdout, "\n", " ", -1))

		if strings.Contains(stdout, disk_store) == false {
			return fmt.Errorf("Disk Store %s does not exist.\nAvailable Disk Stores: %s\n", disk_store, stdout)
		}
	}
	return nil
}

//
//  Create virtual disk
//
func virtualDiskCREATE(c *Config, virtual_disk_disk_store string, virtual_disk_dir string,
	virtual_disk_name string, virtual_disk_size int, virtual_disk_type string) (string, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return virtualDiskCREATE_govmomi(c, virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name, virtual_disk_size, virtual_disk_type)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Println("[virtualDiskCREATE]")

	var virtdisk_id, remote_cmd string
	var err error

	//
	//  Validate disk store exists
	//
	err = diskStoreValidate(c, virtual_disk_disk_store)
	if err != nil {
		return "", fmt.Errorf("Failed to validate disk store: %s\n", err)
	}

	//
	//  Create dir if required
	//
	remote_cmd = fmt.Sprintf("mkdir -p \"/vmfs/volumes/%s/%s\"", virtual_disk_disk_store, virtual_disk_dir)
	_, _ = runRemoteSshCommand(esxiConnInfo, remote_cmd, "create virtual disk dir")

	remote_cmd = fmt.Sprintf("ls -d \"/vmfs/volumes/%s/%s\"", virtual_disk_disk_store, virtual_disk_dir)
	_, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "validate dir exists")
	if err != nil {
		return "", fmt.Errorf("Failed to create virtual disk directory: %s\n", err)
	}

	//
	//  virtdisk_id is just the full path name.
	//
	virtdisk_id = fmt.Sprintf("/vmfs/volumes/%s/%s/%s", virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name)

	//
	//  Validate if it exists already
	//
	remote_cmd = fmt.Sprintf("ls -l \"%s\"", virtdisk_id)
	_, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "validate disk store exists")
	if err == nil {
		log.Println("[virtualDiskCREATE]  Already exists.")
		return virtdisk_id, err
	}

	remote_cmd = fmt.Sprintf("/bin/vmkfstools -c %dG -d %s \"%s\"", virtual_disk_size,
		virtual_disk_type, virtdisk_id)
	_, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Create virtual_disk")
	if err != nil {
		return "", errors.New("Unable to create virtual_disk")
	}

	return virtdisk_id, err
}

//
//  Grow virtual Disk
//
func growVirtualDisk(c *Config, virtdisk_id string, virtdisk_size string) (bool, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return growVirtualDisk_govmomi(c, virtdisk_id, virtdisk_size)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Printf("[growVirtualDisk]\n")

	var didGrowDisk bool
	var newDiskSize int

	_, _, _, currentDiskSize, _, err := virtualDiskREAD(c, virtdisk_id)

	newDiskSize, _ = strconv.Atoi(virtdisk_size)

	log.Printf("[growVirtualDisk] currentDiskSize:%d new_size:%d fullPATH: %s\n", currentDiskSize, newDiskSize, virtdisk_id)

	if currentDiskSize < newDiskSize {
		remote_cmd := fmt.Sprintf("/bin/vmkfstools -X %dG \"%s\"", newDiskSize, virtdisk_id)
		_, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "grow disk")
		if err != nil {
			return didGrowDisk, err
		}
		didGrowDisk = true
	}

	return didGrowDisk, err
}

//
//  Read virtual Disk details
//
func virtualDiskREAD(c *Config, virtdisk_id string) (string, string, string, int, string, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return virtualDiskREAD_govmomi(c, virtdisk_id)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Println("[virtualDiskREAD] Begin")

	var virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name string
	var virtual_disk_type, flatSize string
	var virtual_disk_size int
	var flatSizei64 int64
	var s []string

	//  Split virtdisk_id into it's variables
	s = strings.Split(virtdisk_id, "/")
	log.Printf("[virtualDiskREAD] len=%d cap=%d %v\n", len(s), cap(s), s)
	if len(s) < 6 {
		return "", "", "", 0, "", nil
	} else if len(s) > 6 {
		virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")
	} else {
		virtual_disk_dir = s[4]
	}
	virtual_disk_disk_store = s[3]
	virtual_disk_name = s[len(s)-1]

	// Test if virtual disk exists
	remote_cmd := fmt.Sprintf("test -s \"%s\"", virtdisk_id)
	_, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "test if virtual disk exists")
	if err != nil {
		return "", "", "", 0, "", err
	}

	//  Get virtual disk flat size
	s = strings.Split(virtual_disk_name, ".")
	if len(s) < 2 {
		return "", "", "", 0, "", err
	}
	virtual_disk_nameFlat := fmt.Sprintf("%s-flat.%s", s[0], s[1])

	remote_cmd = fmt.Sprintf("ls -l \"/vmfs/volumes/%s/%s/%s\" | awk '{print $5}'",
		virtual_disk_disk_store, virtual_disk_dir, virtual_disk_nameFlat)
	flatSize, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get size")
	if err != nil {
		return "", "", "", 0, "", err
	}
	flatSizei64, _ = strconv.ParseInt(flatSize, 10, 64)
	virtual_disk_size = int(flatSizei64 / 1024 / 1024 / 1024)

	// Determine virtual disk type  (only works if Guest is powered off)
	remote_cmd = fmt.Sprintf("vmkfstools -t0 \"%s\" |grep -q 'VMFS Z- LVID:' && echo true", virtdisk_id)
	isZeroedThick, _ := runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get disk type.  Is zeroedthick.")

	remote_cmd = fmt.Sprintf("vmkfstools -t0 \"%s\" |grep -q 'VMFS -- LVID:' && echo true", virtdisk_id)
	isEagerZeroedThick, _ := runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get disk type.  Is eagerzeroedthick.")

	remote_cmd = fmt.Sprintf("vmkfstools -t0 \"%s\" |grep -q 'NOMP -- :' && echo true", virtdisk_id)
	isThin, _ := runRemoteSshCommand(esxiConnInfo, remote_cmd, "Get disk type.  Is thin.")

	if isThin == "true" {
		virtual_disk_type = "thin"
	} else if isZeroedThick == "true" {
		virtual_disk_type = "zeroedthick"
	} else if isEagerZeroedThick == "true" {
		virtual_disk_type = "eagerzeroedthick"
	} else {
		virtual_disk_type = "Unknown"
	}

	// Return results
	return virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name, virtual_disk_size, virtual_disk_type, err
}

// ============================================================================
// Govmomi-based Storage Operations
// ============================================================================

// diskStoreValidate_govmomi validates datastore using govmomi
func diskStoreValidate_govmomi(c *Config, disk_store string) error {
	log.Printf("[diskStoreValidate_govmomi]\n")

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return fmt.Errorf("failed to get govmomi client: %w", err)
	}

	// Try to find the datastore
	ds, err := getDatastoreByName(gc.Context(), gc.Finder, disk_store)
	if err != nil {
		// Try rescanning and search again
		host, err := getHostSystem(gc.Context(), gc.Finder)
		if err == nil {
			hostStorageSystem, err := host.ConfigManager().StorageSystem(gc.Context())
			if err == nil {
				_ = hostStorageSystem.RescanAllHba(gc.Context())
				_ = hostStorageSystem.RescanVmfs(gc.Context())
			}
		}

		// Try again after rescan
		ds, err = getDatastoreByName(gc.Context(), gc.Finder, disk_store)
		if err != nil {
			return fmt.Errorf("disk store %s does not exist: %w", disk_store, err)
		}
	}

	// Check if accessible
	accessible, err := isDatastoreAccessible(gc.Context(), ds)
	if err != nil {
		return fmt.Errorf("failed to check datastore accessibility: %w", err)
	}

	if !accessible {
		return fmt.Errorf("disk store %s is not accessible", disk_store)
	}

	log.Printf("[diskStoreValidate_govmomi] Datastore %s is valid and accessible\n", disk_store)
	return nil
}

// virtualDiskCREATE_govmomi creates a virtual disk using govmomi
func virtualDiskCREATE_govmomi(c *Config, virtual_disk_disk_store string, virtual_disk_dir string,
	virtual_disk_name string, virtual_disk_size int, virtual_disk_type string) (string, error) {
	log.Println("[virtualDiskCREATE_govmomi]")

	// Validate disk store exists
	err := diskStoreValidate_govmomi(c, virtual_disk_disk_store)
	if err != nil {
		return "", fmt.Errorf("failed to validate disk store: %w", err)
	}

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
	if err != nil {
		return "", fmt.Errorf("failed to get datastore: %w", err)
	}

	// Build full virtual disk path
	virtdisk_id := fmt.Sprintf("/vmfs/volumes/%s/%s/%s", virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name)
	diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

	// Check if directory exists by trying to create the virtual disk
	// govmomi VirtualDiskManager will handle directory creation if needed

	// Map disk type to govmomi backing
	var diskType string

	switch virtual_disk_type {
	case "thin":
		diskType = string(types.VirtualDiskTypeThin)
	case "zeroedthick":
		diskType = string(types.VirtualDiskTypeThick)
	case "eagerzeroedthick":
		diskType = string(types.VirtualDiskTypeEagerZeroedThick)
	default:
		diskType = string(types.VirtualDiskTypeThin)
	}

	// Create disk spec
	spec := &types.FileBackedVirtualDiskSpec{
		VirtualDiskSpec: types.VirtualDiskSpec{
			DiskType:    diskType,
			AdapterType: string(types.VirtualDiskAdapterTypeLsiLogic),
		},
		CapacityKb: int64(virtual_disk_size * 1024 * 1024), // Convert GB to KB
	}

	// Create virtual disk manager
	dm := object.NewVirtualDiskManager(gc.Client.Client)

	// Create the virtual disk (datacenter can be nil for standalone ESXi)
	task, err := dm.CreateVirtualDisk(gc.Context(), diskPath, nil, spec)
	if err != nil {
		return "", fmt.Errorf("failed to create virtual disk: %w", err)
	}

	err = waitForTask(gc.Context(), task)
	if err != nil {
		return "", fmt.Errorf("failed to create virtual disk: %w", err)
	}

	log.Printf("[virtualDiskCREATE_govmomi] Created virtual disk: %s\n", virtdisk_id)
	return virtdisk_id, nil
}

// virtualDiskREAD_govmomi reads virtual disk properties using govmomi
func virtualDiskREAD_govmomi(c *Config, virtdisk_id string) (string, string, string, int, string, error) {
	log.Println("[virtualDiskREAD_govmomi] Begin")

	var virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name string
	var virtual_disk_type string
	var virtual_disk_size int

	// Split virtdisk_id into its variables
	s := strings.Split(virtdisk_id, "/")
	log.Printf("[virtualDiskREAD_govmomi] len=%d cap=%d %v\n", len(s), cap(s), s)
	if len(s) < 6 {
		return "", "", "", 0, "", nil
	} else if len(s) > 6 {
		virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")
	} else {
		virtual_disk_dir = s[4]
	}
	virtual_disk_disk_store = s[3]
	virtual_disk_name = s[len(s)-1]

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", "", "", 0, "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
	if err != nil {
		return "", "", "", 0, "", fmt.Errorf("failed to get datastore: %w", err)
	}

	diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

	// Query virtual disk
	dm := object.NewVirtualDiskManager(gc.Client.Client)
	uuid, err := dm.QueryVirtualDiskUuid(gc.Context(), diskPath, nil)
	if err != nil {
		return "", "", "", 0, "", fmt.Errorf("virtual disk not found: %w", err)
	}

	log.Printf("[virtualDiskREAD_govmomi] Found disk with UUID: %s\n", uuid)

	// Get disk size from the -flat.vmdk file
	// Parse the virtual disk name to construct the flat file name
	parts := strings.Split(virtual_disk_name, ".")
	if len(parts) >= 2 {
		virtual_disk_nameFlat := fmt.Sprintf("%s-flat.%s", parts[0], parts[1])

		// Use the datastore browser to search for the flat file
		browser, err := ds.Browser(gc.Context())
		if err == nil {
			spec := types.HostDatastoreBrowserSearchSpec{
				MatchPattern: []string{virtual_disk_nameFlat},
				Details: &types.FileQueryFlags{
					FileSize: true,
				},
			}

			searchPath := ds.Path(virtual_disk_dir)
			task, err := browser.SearchDatastore(gc.Context(), searchPath, &spec)
			if err == nil {
				info, err := task.WaitForResult(gc.Context())
				if err == nil {
					if result, ok := info.Result.(types.HostDatastoreBrowserSearchResults); ok {
						if len(result.File) > 0 {
							if fileInfo, ok := result.File[0].(*types.FileInfo); ok {
								// Convert bytes to GB
								virtual_disk_size = int(fileInfo.FileSize / (1024 * 1024 * 1024))
								log.Printf("[virtualDiskREAD_govmomi] Disk size: %d GB\n", virtual_disk_size)
							}
						}
					}
				}
			}
		}
	}

	// For disk type, we'll default to "Unknown" for now
	// The disk type requires parsing the VMDK descriptor or using vmkfstools
	virtual_disk_type = "Unknown"

	// Return results
	return virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name, virtual_disk_size, virtual_disk_type, err
}

// growVirtualDisk_govmomi grows a virtual disk using govmomi
func growVirtualDisk_govmomi(c *Config, virtdisk_id string, virtdisk_size string) (bool, error) {
	log.Printf("[growVirtualDisk_govmomi]\n")

	var didGrowDisk bool
	var newDiskSize int

	// Get current disk size
	_, _, _, currentDiskSize, _, err := virtualDiskREAD_govmomi(c, virtdisk_id)
	if err != nil {
		return didGrowDisk, fmt.Errorf("failed to read current disk size: %w", err)
	}

	newDiskSize, _ = strconv.Atoi(virtdisk_size)

	log.Printf("[growVirtualDisk_govmomi] currentDiskSize:%d new_size:%d fullPATH: %s\n", currentDiskSize, newDiskSize, virtdisk_id)

	if currentDiskSize < newDiskSize {
		// Parse the virtdisk_id to get datastore and path
		s := strings.Split(virtdisk_id, "/")
		if len(s) < 6 {
			return didGrowDisk, fmt.Errorf("invalid virtdisk_id format")
		}

		var virtual_disk_dir string
		if len(s) > 6 {
			virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")
		} else {
			virtual_disk_dir = s[4]
		}
		virtual_disk_disk_store := s[3]
		virtual_disk_name := s[len(s)-1]

		gc, err := c.GetGovmomiClient()
		if err != nil {
			return didGrowDisk, fmt.Errorf("failed to get govmomi client: %w", err)
		}

		ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
		if err != nil {
			return didGrowDisk, fmt.Errorf("failed to get datastore: %w", err)
		}

		diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

		// Create virtual disk manager
		dm := object.NewVirtualDiskManager(gc.Client.Client)

		// Extend the virtual disk
		newCapacityKb := int64(newDiskSize * 1024 * 1024) // Convert GB to KB
		task, err := dm.ExtendVirtualDisk(gc.Context(), diskPath, nil, newCapacityKb, nil)
		if err != nil {
			return didGrowDisk, fmt.Errorf("failed to extend virtual disk: %w", err)
		}

		err = waitForTask(gc.Context(), task)
		if err != nil {
			return didGrowDisk, fmt.Errorf("failed to extend virtual disk: %w", err)
		}

		didGrowDisk = true
		log.Printf("[growVirtualDisk_govmomi] Successfully grew disk to %d GB\n", newDiskSize)
	}

	return didGrowDisk, nil
}
