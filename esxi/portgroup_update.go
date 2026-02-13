package esxi

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePORTGROUPUpdate(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourcePORTGROUPUpdate]")

	name := d.Get("name").(string)
	vlan := d.Get("vlan").(int)

	// Validate security policies
	promiscuous_mode := d.Get("promiscuous_mode").(string)
	if promiscuous_mode != "true" && promiscuous_mode != "false" && promiscuous_mode != "" {
		return errors.New("Error: promiscuous_mode must be true, false or '' to inherit")
	}

	forged_transmits := d.Get("forged_transmits").(string)
	if forged_transmits != "true" && forged_transmits != "false" && forged_transmits != "" {
		return errors.New("Error: forged_transmits must be true, false or '' to inherit")
	}

	mac_changes := d.Get("mac_changes").(string)
	if mac_changes != "true" && mac_changes != "false" && mac_changes != "" {
		return errors.New("Error: mac_changes must be true, false or '' to inherit")
	}

	err := portgroupUpdate_govmomi(c, name, vlan, promiscuous_mode, forged_transmits, mac_changes)
	if err != nil {
		return fmt.Errorf("Failed to update portgroup: %s\n", err)
	}

	// Refresh
	return resourcePORTGROUPRead(d, m)
}
