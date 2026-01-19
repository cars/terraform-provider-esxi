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

	if c.useGovmomi {
		// Use govmomi to delete resource pool
		err := resourcePoolDelete_govmomi(c, pool_id)
		if err != nil {
			log.Printf("[resourceRESOURCEPOOLDelete] Failed destroy resource pool id: %s\n", pool_id)
			return fmt.Errorf("Failed to delete pool: %s\n", err)
		}
	} else {
		// Fallback to SSH
		esxiConnInfo := getConnectionInfo(c)
		var remote_cmd, stdout string
		var err error

		remote_cmd = fmt.Sprintf("vim-cmd hostsvc/rsrc/destroy %s", pool_id)
		stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "destroy resource pool")
		if err != nil {
			log.Printf("[resourceRESOURCEPOOLDelete] Failed destroy resource pool id: %s\n", stdout)
			return fmt.Errorf("Failed to delete pool: %s\n", err)
		}
	}

	d.SetId("")
	return nil
}
