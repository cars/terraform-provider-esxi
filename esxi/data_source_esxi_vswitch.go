package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceVswitch() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVswitchRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the virtual switch to look up.",
			},
			"ports": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of ports on the virtual switch.",
			},
			"mtu": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "MTU of the virtual switch.",
			},
			"link_discovery_mode": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Link Discovery Mode of the virtual switch.",
			},
			"promiscuous_mode": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Promiscuous mode (true=Accept/false=Reject).",
			},
			"mac_changes": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "MAC address changes (true=Accept/false=Reject).",
			},
			"forged_transmits": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Forged transmits (true=Accept/false=Reject).",
			},
			"uplink": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Description: "List of uplinks for the virtual switch.",
			},
		},
	}
}

func dataSourceVswitchRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[dataSourceVswitchRead]")

	name := d.Get("name").(string)
	if name == "" {
		return fmt.Errorf("Virtual switch name is required")
	}

	// Read virtual switch configuration
	ports, mtu, uplinks, linkDiscoveryMode, promiscuousMode, macChanges, forgedTransmits, err := vswitchRead(c, name)
	if err != nil {
		return fmt.Errorf("Failed to read virtual switch '%s': %s", name, err)
	}

	// Set the ID to the vswitch name
	d.SetId(name)

	// Set computed fields
	d.Set("name", name)
	d.Set("ports", ports)
	d.Set("mtu", mtu)
	d.Set("link_discovery_mode", linkDiscoveryMode)
	d.Set("promiscuous_mode", promiscuousMode)
	d.Set("mac_changes", macChanges)
	d.Set("forged_transmits", forgedTransmits)

	// Convert uplinks to list format
	uplinkList := make([]map[string]interface{}, len(uplinks))
	for i, uplink := range uplinks {
		uplinkList[i] = map[string]interface{}{
			"name": uplink,
		}
	}
	d.Set("uplink", uplinkList)

	log.Printf("[dataSourceVswitchRead] Successfully read virtual switch '%s'", name)
	return nil
}