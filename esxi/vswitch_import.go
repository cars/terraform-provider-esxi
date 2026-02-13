package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceVSWITCHImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*Config)
	log.Println("[resourceVSWITCHImport]")

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	// Use govmomi to verify vswitch exists
	_, _, _, _, _, _, _, err := vswitchRead(c, d.Id())
	if err != nil {
		return results, fmt.Errorf("Failed to import vswitch '%s': %s", d.Id(), err)
	}

	d.SetId(d.Id())
	d.Set("name", d.Id())

	return results, nil
}
