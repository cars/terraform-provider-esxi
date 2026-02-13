package esxi

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRESOURCEPOOLCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceRESOURCEPOOLCreate]")

	var pool_id, parent_pool string
	var err error

	resource_pool_name := d.Get("resource_pool_name").(string)
	cpu_min := d.Get("cpu_min").(int)
	cpu_min_expandable := d.Get("cpu_min_expandable").(string)
	cpu_max := d.Get("cpu_max").(int)
	cpu_shares := strings.ToLower(d.Get("cpu_shares").(string))
	mem_min := d.Get("mem_min").(int)
	mem_min_expandable := d.Get("mem_min_expandable").(string)
	mem_max := d.Get("mem_max").(int)
	mem_shares := strings.ToLower(d.Get("mem_shares").(string))
	parent_pool = "Resources"

	if resource_pool_name == string('/') {
		return fmt.Errorf("Missing required resource_pool_name")
	}

	if resource_pool_name[0] == '/' {
		return fmt.Errorf("Resource Pool Name cannot start with /")
	}
	i := strings.LastIndex(resource_pool_name, "/")
	if i > 2 {
		parent_pool = resource_pool_name[:i]
		resource_pool_name = resource_pool_name[i+1:]
	}

	//  Check if already exists
	stdout, _ := getPoolID(c, resource_pool_name)
	if stdout != "" {
		d.SetId(stdout)
		return nil
	}

	pool_id, err = resourcePoolCreate_govmomi(c, resource_pool_name, cpu_min, cpu_min_expandable,
		cpu_max, cpu_shares, mem_min, mem_min_expandable, mem_max, mem_shares, parent_pool)
	if err != nil {
		d.SetId("")
		return fmt.Errorf("Failed to create pool: %s\n", err)
	}

	//  Set pool_id
	d.SetId(pool_id)

	// Refresh
	resource_pool_name, cpu_min, cpu_min_expandable, cpu_max, cpu_shares, mem_min, mem_min_expandable, mem_max, mem_shares, err = resourcePoolRead(c, pool_id)
	if err != nil {
		d.SetId("")
		return fmt.Errorf("Failed to refresh pool: %s\n", err)
	}

	d.Set("resource_pool_name", resource_pool_name)
	d.Set("cpu_min", cpu_min)
	d.Set("cpu_min_expandable", cpu_min_expandable)
	d.Set("cpu_max", cpu_max)
	d.Set("cpu_shares", cpu_shares)
	d.Set("mem_min", mem_min)
	d.Set("mem_min_expandable", mem_min_expandable)
	d.Set("mem_max", mem_max)
	d.Set("mem_shares", mem_shares)

	return nil
}
