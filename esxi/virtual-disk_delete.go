package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceVIRTUALDISKDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceVIRTUALDISKDelete]")

	virtdisk_id := d.Id()

	err := virtualDiskDelete_govmomi(c, virtdisk_id)
	if err != nil {
		return fmt.Errorf("Failed to delete virtual disk: %w", err)
	}

	d.SetId("")
	return nil
}
