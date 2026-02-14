package esxi

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
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

// findVirtualDiskInDir searches for .vmdk files using govmomi API
func findVirtualDiskInDir(c *Config, diskStore, dir string) (string, error) {
	log.Printf("[findVirtualDiskInDir]")

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	ds, err := getDatastoreByName(gc.Context(), gc.Finder, diskStore)
	if err != nil {
		return "", fmt.Errorf("failed to get datastore: %w", err)
	}

	// Get datastore browser
	browser, err := ds.Browser(gc.Context())
	if err != nil {
		return "", fmt.Errorf("failed to get datastore browser: %w", err)
	}

	// Create search spec for .vmdk files
	spec := types.HostDatastoreBrowserSearchSpec{
		MatchPattern: []string{"*.vmdk"},
		Details: &types.FileQueryFlags{
			FileSize: false,
		},
	}

	// Search the directory
	searchPath := ds.Path(dir)
	task, err := browser.SearchDatastore(gc.Context(), searchPath, &spec)
	if err != nil {
		return "", fmt.Errorf("failed to search datastore: %w", err)
	}

	info, err := task.WaitForResult(gc.Context())
	if err != nil {
		return "", fmt.Errorf("failed to get search results: %w", err)
	}

	result, ok := info.Result.(types.HostDatastoreBrowserSearchResults)
	if !ok {
		return "", fmt.Errorf("unexpected search result type")
	}

	// Iterate through files, skip -flat.vmdk files, return first descriptor .vmdk
	for _, file := range result.File {
		if fileInfo, ok := file.(*types.FileInfo); ok {
			filename := fileInfo.Path
			// Skip -flat.vmdk files
			if strings.HasSuffix(filename, "-flat.vmdk") {
				continue
			}
			// Return first descriptor .vmdk file
			if strings.HasSuffix(filename, ".vmdk") {
				log.Printf("[findVirtualDiskInDir] Found virtual disk: %s\n", filename)
				return filename, nil
			}
		}
	}

	// No vmdk found - return empty string (not error)
	log.Printf("[findVirtualDiskInDir] No virtual disk found in directory\n")
	return "", nil
}