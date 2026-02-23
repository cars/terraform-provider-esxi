package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceVSWITCHDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceVSWITCHDelete]")

	name := d.Id()

	err := vswitchDelete(c, name)
	if err != nil {
		log.Printf("[resourceVSWITCHDelete] Failed destroy vswitch: %s\n", err)
		return fmt.Errorf("Failed to destroy vswitch: %s\n", err)
	}

	d.SetId("")
	return nil
}
