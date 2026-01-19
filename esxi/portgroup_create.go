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
	if c.useGovmomi {
		err := portgroupCreate_govmomi(c, name, vswitch)
		if err != nil {
			d.SetId("")
			return fmt.Errorf("Failed to add portgroup: %s\n", err)
		}
	} else {
		esxiConnInfo := getConnectionInfo(c)
		var stdout string
		var remote_cmd string
		var err error

		remote_cmd = fmt.Sprintf("esxcli network vswitch standard portgroup add -v \"%s\" -p \"%s\"",
			vswitch, name)

		stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "create portgroup")
		if err != nil {
			d.SetId("")
			return fmt.Errorf("Failed to add portgroup: %s\n%s\n", stdout, err)
		}
	}

	//  Set id
	d.SetId(name)

	return resourcePORTGROUPUpdate(d, m)
}
