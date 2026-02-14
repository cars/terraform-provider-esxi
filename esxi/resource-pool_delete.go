package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRESOURCEPOOLDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceRESOURCEPOOLDelete]")

	pool_id := d.Id()

	err := resourcePoolDelete(c, pool_id)
	if err != nil {
		log.Printf("[resourceRESOURCEPOOLDelete] Failed destroy resource pool id: %s\n", pool_id)
		return fmt.Errorf("Failed to delete pool: %s\n", err)
	}

	d.SetId("")
	return nil
}
