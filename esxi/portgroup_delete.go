package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePORTGROUPDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourcePORTGROUPDelete]")

	name := d.Id()

	err := portgroupDelete_govmomi(c, name)
	if err != nil {
		log.Printf("[resourcePORTGROUPDelete] Failed destroy PORTGROUP: %s\n", err)
		return fmt.Errorf("Failed to destroy portgroup: %s\n", err)
	}

	d.SetId("")
	return nil
}
