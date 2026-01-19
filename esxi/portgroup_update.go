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

	if c.useGovmomi {
		err := portgroupUpdate_govmomi(c, name, vlan, promiscuous_mode, forged_transmits, mac_changes)
		if err != nil {
			return fmt.Errorf("Failed to update portgroup: %s\n", err)
		}
	} else {
		esxiConnInfo := getConnectionInfo(c)
		var stdout string
		var remote_cmd string
		var err error
		var promiscuous_mode_cmd, forged_transmits_cmd, mac_changes_cmd string

		//  set vlan id
		remote_cmd = fmt.Sprintf("esxcli network vswitch standard portgroup set -v \"%d\" -p \"%s\"",
			vlan, name)

		stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "portgroup set vlan")
		if err != nil {
			d.SetId("") // TODO do we really want to do this? maybe only if the portgroup
			return fmt.Errorf("Failed to set portgroup: %s\n%s\n", stdout, err)
		}

		// set the security policies.
		if promiscuous_mode != "" {
			promiscuous_mode_cmd = fmt.Sprintf("--allow-promiscuous=%s", promiscuous_mode)
		}

		if forged_transmits != "" {
			forged_transmits_cmd = fmt.Sprintf("--allow-forged-transmits=%s", forged_transmits)
		}

		if mac_changes != "" {
			mac_changes_cmd = fmt.Sprintf("--allow-mac-change=%s", mac_changes)
		}

		// There is no way to set any param to inherited, so we must use the -u to set inherited for all three params..
		remote_cmd = fmt.Sprintf("esxcli network vswitch standard portgroup policy security set -p \"%s\" -u %s %s %s", name, promiscuous_mode_cmd, forged_transmits_cmd, mac_changes_cmd)
		stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "set the inherited portgroup security policy")
		if err != nil {
			return fmt.Errorf("Failed to set the inherited portgroup security policy: %s\n%s\n", stdout, err)
		}
	}

	// Refresh
	return resourcePORTGROUPRead(d, m)
}
