package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceResourcePool() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceResourcePoolRead,

		Schema: map[string]*schema.Schema{
			"resource_pool_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the resource pool to look up.",
			},
			"cpu_min": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "CPU minimum (in MHz).",
			},
			"cpu_min_expandable": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Can pool borrow CPU resources from parent?",
			},
			"cpu_max": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "CPU maximum (in MHz).",
			},
			"cpu_shares": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CPU shares (low/normal/high/<custom>).",
			},
			"mem_min": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Memory minimum (in MB).",
			},
			"mem_min_expandable": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Can pool borrow memory resources from parent?",
			},
			"mem_max": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Memory maximum (in MB).",
			},
			"mem_shares": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Memory shares (low/normal/high/<custom>).",
			},
		},
	}
}

func dataSourceResourcePoolRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[dataSourceResourcePoolRead]")

	name := d.Get("resource_pool_name").(string)
	if name == "" {
		return fmt.Errorf("Resource pool name is required")
	}

	// Get resource pool ID first
	poolID, err := getPoolID(c, name)
	if err != nil {
		return fmt.Errorf("Failed to find resource pool '%s': %s", name, err)
	}

	// Read resource pool configuration
	_, cpuMin, cpuMinExpandable, cpuMax, cpuShares, memMin, memMinExpandable, memMax, memShares, err := resourcePoolRead(c, poolID)
	if err != nil {
		return fmt.Errorf("Failed to read resource pool '%s': %s", name, err)
	}

	// Set the ID to the resource pool ID
	d.SetId(poolID)

	// Set computed fields
	d.Set("resource_pool_name", name)
	d.Set("cpu_min", cpuMin)
	d.Set("cpu_min_expandable", cpuMinExpandable)
	d.Set("cpu_max", cpuMax)
	d.Set("cpu_shares", cpuShares)
	d.Set("mem_min", memMin)
	d.Set("mem_min_expandable", memMinExpandable)
	d.Set("mem_max", memMax)
	d.Set("mem_shares", memShares)

	log.Printf("[dataSourceResourcePoolRead] Successfully read resource pool '%s'", name)
	return nil
}