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

	if c.useGovmomi {
		err := portgroupDelete_govmomi(c, name)
		if err != nil {
			log.Printf("[resourcePORTGROUPDelete] Failed destroy PORTGROUP: %s\n", err)
			return fmt.Errorf("Failed to destroy portgroup: %s\n", err)
		}
	} else {
		esxiConnInfo := getConnectionInfo(c)
		var remote_cmd, stdout string
		var err error

		vswitch := d.Get("vswitch").(string)

		remote_cmd = fmt.Sprintf("esxcli network vswitch standard portgroup remove -v \"%s\" -p \"%s\"",
			vswitch, name)

		stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "destroy PORTGROUP")
		if err != nil {
			log.Printf("[resourcePORTGROUPDelete] Failed destroy PORTGROUP: %s\n", stdout)
			return fmt.Errorf("Failed to destroy portgroup: %s\n%s\n", stdout, err)
		}
	}

	d.SetId("")
	return nil
}
