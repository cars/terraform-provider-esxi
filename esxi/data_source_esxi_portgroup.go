package esxi

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePortgroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePortgroupRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the port group to look up.",
			},
			"vswitch": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The vswitch name where the port group is located.",
			},
			"vlan": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The VLAN ID of the port group.",
			},
			"promiscuous_mode": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Promiscuous mode (true=Accept/false=Reject).",
			},
			"mac_changes": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "MAC address changes (true=Accept/false=Reject).",
			},
			"forged_transmits": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Forged transmits (true=Accept/false=Reject).",
			},
		},
	}
}

func dataSourcePortgroupRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[dataSourcePortgroupRead]")

	name := d.Get("name").(string)
	if name == "" {
		return fmt.Errorf("Port group name is required")
	}

	// Read port group basic info
	vswitch, vlan, err := portgroupRead(c, name)
	if err != nil {
		return fmt.Errorf("Failed to read port group '%s': %s", name, err)
	}

	// Read security policy
	policy, err := portgroupSecurityPolicyRead(c, name)
	if err != nil {
		log.Printf("[dataSourcePortgroupRead] Warning: failed to read security policy: %s", err)
		policy = &portgroupSecurityPolicy{}
	}

	// Set the ID to the port group name
	d.SetId(name)

	// Set computed fields
	d.Set("name", name)
	d.Set("vswitch", vswitch)
	d.Set("vlan", vlan)

	// Convert boolean security policy to string values
	d.Set("promiscuous_mode", boolToString(policy.AllowPromiscuous))
	d.Set("mac_changes", boolToString(policy.AllowMACAddressChange))
	d.Set("forged_transmits", boolToString(policy.AllowForgedTransmits))

	log.Printf("[dataSourcePortgroupRead] Successfully read port group '%s'", name)
	return nil
}

// boolToString converts boolean to "true"/"false" string
func boolToString(b bool) string {
	return strconv.FormatBool(b)
}