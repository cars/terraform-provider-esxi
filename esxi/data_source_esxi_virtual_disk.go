package esxi

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVirtualDiskRead,

		Schema: map[string]*schema.Schema{
			"virtual_disk_disk_store": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The disk store where the virtual disk is located.",
			},
			"virtual_disk_dir": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The directory where the virtual disk is located.",
			},
			"virtual_disk_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the virtual disk to look up. If not specified, will look for .vmdk files in the directory.",
			},
			"virtual_disk_size": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Virtual disk size in GB.",
			},
			"virtual_disk_type": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Virtual disk type (thin, zeroedthick, eagerzeroedthick).",
			},
		},
	}
}

func dataSourceVirtualDiskRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[dataSourceVirtualDiskRead]")

	diskStore := d.Get("virtual_disk_disk_store").(string)
	dir := d.Get("virtual_disk_dir").(string)
	diskName := d.Get("virtual_disk_name").(string)

	if diskStore == "" {
		return fmt.Errorf("Disk store is required")
	}
	if dir == "" {
		return fmt.Errorf("Directory is required")
	}

	// Validate disk store exists
	err := diskStoreValidate(c, diskStore)
	if err != nil {
		return fmt.Errorf("Invalid disk store '%s': %s", diskStore, err)
	}

	// If disk name not provided, try to find a .vmdk file in the directory
	if diskName == "" {
		diskName, err = findVirtualDiskInDir(c, diskStore, dir)
		if err != nil {
			return fmt.Errorf("Failed to find virtual disk in directory '%s': %s", dir, err)
		}
		if diskName == "" {
			return fmt.Errorf("No virtual disk found in directory '%s'", dir)
		}
	}

	// Construct the virtual disk ID (same format as resource)
	virtdiskID := fmt.Sprintf("%s/%s/%s", diskStore, dir, diskName)

	// Read virtual disk configuration
	readDiskStore, readDir, readName, size, diskType, err := virtualDiskREAD(c, virtdiskID)
	if err != nil {
		return fmt.Errorf("Failed to read virtual disk '%s': %s", virtdiskID, err)
	}

	// Set the ID to the virtual disk path
	d.SetId(virtdiskID)

	// Set computed fields
	d.Set("virtual_disk_disk_store", readDiskStore)
	d.Set("virtual_disk_dir", readDir)
	d.Set("virtual_disk_name", readName)
	d.Set("virtual_disk_size", size)
	if diskType != "Unknown" {
		d.Set("virtual_disk_type", diskType)
	}

	log.Printf("[dataSourceVirtualDiskRead] Successfully read virtual disk '%s'", virtdiskID)
	return nil
}

// findVirtualDiskInDir searches for .vmdk files in the specified directory
func findVirtualDiskInDir(c *Config, diskStore, dir string) (string, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return findVirtualDiskInDir_govmomi(c, diskStore, dir)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Printf("[findVirtualDiskInDir]")

	remote_cmd := fmt.Sprintf("ls \"/vmfs/volumes/%s/%s\"/*.vmdk 2>/dev/null | head -1",
		diskStore, dir)
	
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "find virtual disk")
	if err != nil {
		return "", err
	}

	if stdout == "" {
		return "", nil
	}

	// Extract just the filename from the full path
	filename := filepath.Base(strings.TrimSpace(stdout))
	
	// Remove -flat.vmdk suffix if present, return the descriptor .vmdk
	if strings.HasSuffix(filename, "-flat.vmdk") {
		filename = strings.Replace(filename, "-flat.vmdk", ".vmdk", 1)
	}

	return filename, nil
}

// findVirtualDiskInDir_govmomi searches for .vmdk files using govmomi API
func findVirtualDiskInDir_govmomi(c *Config, diskStore, dir string) (string, error) {
	// TODO: Implement govmomi-based directory listing for virtual disks
	// This requires using the datastore browser API to enumerate .vmdk files
	return "", fmt.Errorf("findVirtualDiskInDir_govmomi not yet implemented - use SSH mode or specify virtual_disk_name explicitly")
}