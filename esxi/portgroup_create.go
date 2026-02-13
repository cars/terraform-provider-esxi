package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePORTGROUPCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourcePORTGROUPCreate]")

	name := d.Get("name").(string)
	vswitch := d.Get("vswitch").(string)

	//  Create PORTGROUP
	err := portgroupCreate_govmomi(c, name, vswitch)
	if err != nil {
		d.SetId("")
		return fmt.Errorf("Failed to add portgroup: %s\n", err)
	}

	//  Set id
	d.SetId(name)

	return resourcePORTGROUPUpdate(d, m)
}
