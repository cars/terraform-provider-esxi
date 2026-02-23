package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePORTGROUPImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*Config)
	log.Println("[resourcePORTGROUPImport]")

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	// Use govmomi to verify portgroup exists
	_, _, err := portgroupRead(c, d.Id())
	if err != nil {
		return results, fmt.Errorf("Failed to import portgroup '%s': %s", d.Id(), err)
	}

	d.SetId(d.Id())
	d.Set("name", d.Id())

	return results, nil
}
