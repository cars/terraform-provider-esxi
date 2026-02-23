package esxi

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// ============================================================================
// Resource Pool Operations
// ============================================================================

// getPoolID gets resource pool ID by name using govmomi
func getPoolID(c *Config, resource_pool_name string) (string, error) {
	log.Printf("[getPoolID] Getting pool ID for: %s\n", resource_pool_name)

	if resource_pool_name == "/" || resource_pool_name == "Resources" {
		return "ha-root-pool", nil
	}

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	rootPool, err := getRootResourcePool(gc.Context(), gc.Finder)
	if err != nil {
		return "", fmt.Errorf("failed to get root resource pool: %w", err)
	}

	// Extract just the pool name from path
	result := strings.Split(resource_pool_name, "/")
	poolName := result[len(result)-1]

	// Find the pool by path
	pool, err := findResourcePoolByPath(gc.Context(), rootPool, resource_pool_name)
	if err != nil {
		return "", err
	}

	// Verify the pool name matches
	var poolMo mo.ResourcePool
	err = pool.Properties(gc.Context(), pool.Reference(), []string{"name"}, &poolMo)
	if err != nil {
		return "", fmt.Errorf("failed to get pool properties: %w", err)
	}

	if poolMo.Name != poolName && poolName != "Resources" {
		return "", fmt.Errorf("pool name mismatch: expected %s, got %s", poolName, poolMo.Name)
	}

	return pool.Reference().Value, nil
}

// getPoolNAME gets resource pool name by ID using govmomi
func getPoolNAME(c *Config, resource_pool_id string) (string, error) {
	log.Printf("[getPoolNAME] Getting pool name for ID: %s\n", resource_pool_id)

	if resource_pool_id == "ha-root-pool" {
		return "/", nil
	}

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	// Create resource pool object from reference
	poolRef := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: resource_pool_id,
	}
	pool := object.NewResourcePool(gc.Client.Client, poolRef)

	// Get the full path by traversing up the hierarchy
	var fullPath string
	currentPool := pool

	for {
		var poolMo mo.ResourcePool
		err := currentPool.Properties(gc.Context(), currentPool.Reference(), []string{"name", "parent"}, &poolMo)
		if err != nil {
			return "", fmt.Errorf("failed to get pool properties: %w", err)
		}

		// Prepend current pool name to path
		if poolMo.Name != "Resources" {
			if fullPath == "" {
				fullPath = poolMo.Name
			} else {
				fullPath = poolMo.Name + "/" + fullPath
			}
		}

		// Check if we've reached the root
		if poolMo.Parent == nil || poolMo.Parent.Type != "ResourcePool" {
			break
		}

		// Move to parent
		currentPool = object.NewResourcePool(gc.Client.Client, *poolMo.Parent)
	}

	return fullPath, nil
}

// resourcePoolRead reads resource pool configuration using govmomi
func resourcePoolRead(c *Config, pool_id string) (string, int, string, int, string, int, string, int, string, error) {
	log.Printf("[resourcePoolRead] Reading pool ID: %s\n", pool_id)

	var cpu_min, cpu_max, mem_min, mem_max int
	var cpu_shares, mem_shares string
	var cpu_min_expandable, mem_min_expandable string

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", 0, "", 0, "", 0, "", 0, "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	// Create resource pool object from reference
	poolRef := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: pool_id,
	}
	pool := object.NewResourcePool(gc.Client.Client, poolRef)

	// Get pool configuration
	var poolMo mo.ResourcePool
	err = pool.Properties(gc.Context(), pool.Reference(), []string{"name", "config"}, &poolMo)
	if err != nil {
		return "", 0, "", 0, "", 0, "", 0, "", fmt.Errorf("failed to get pool properties: %w", err)
	}

	// Extract CPU allocation
	if poolMo.Config.CpuAllocation.Reservation != nil {
		cpu_min = int(*poolMo.Config.CpuAllocation.Reservation)
	}
	if poolMo.Config.CpuAllocation.ExpandableReservation != nil {
		cpu_min_expandable = fmt.Sprintf("%t", *poolMo.Config.CpuAllocation.ExpandableReservation)
	} else {
		cpu_min_expandable = "true"
	}
	if poolMo.Config.CpuAllocation.Limit != nil && *poolMo.Config.CpuAllocation.Limit >= 0 {
		cpu_max = int(*poolMo.Config.CpuAllocation.Limit)
	}
	if poolMo.Config.CpuAllocation.Shares != nil {
		if poolMo.Config.CpuAllocation.Shares.Level == types.SharesLevelNormal ||
			poolMo.Config.CpuAllocation.Shares.Level == types.SharesLevelLow ||
			poolMo.Config.CpuAllocation.Shares.Level == types.SharesLevelHigh {
			cpu_shares = string(poolMo.Config.CpuAllocation.Shares.Level)
		} else if poolMo.Config.CpuAllocation.Shares.Level == types.SharesLevelCustom {
			cpu_shares = fmt.Sprintf("%d", poolMo.Config.CpuAllocation.Shares.Shares)
		}
	}

	// Extract Memory allocation
	if poolMo.Config.MemoryAllocation.Reservation != nil {
		mem_min = int(*poolMo.Config.MemoryAllocation.Reservation)
	}
	if poolMo.Config.MemoryAllocation.ExpandableReservation != nil {
		mem_min_expandable = fmt.Sprintf("%t", *poolMo.Config.MemoryAllocation.ExpandableReservation)
	} else {
		mem_min_expandable = "true"
	}
	if poolMo.Config.MemoryAllocation.Limit != nil && *poolMo.Config.MemoryAllocation.Limit >= 0 {
		mem_max = int(*poolMo.Config.MemoryAllocation.Limit)
	}
	if poolMo.Config.MemoryAllocation.Shares != nil {
		if poolMo.Config.MemoryAllocation.Shares.Level == types.SharesLevelNormal ||
			poolMo.Config.MemoryAllocation.Shares.Level == types.SharesLevelLow ||
			poolMo.Config.MemoryAllocation.Shares.Level == types.SharesLevelHigh {
			mem_shares = string(poolMo.Config.MemoryAllocation.Shares.Level)
		} else if poolMo.Config.MemoryAllocation.Shares.Level == types.SharesLevelCustom {
			mem_shares = fmt.Sprintf("%d", poolMo.Config.MemoryAllocation.Shares.Shares)
		}
	}

	// Get pool name
	resource_pool_name, err := getPoolNAME(c, pool_id)
	if err != nil {
		return "", 0, "", 0, "", 0, "", 0, "", fmt.Errorf("failed to get pool name: %w", err)
	}

	log.Printf("[resourcePoolRead] Successfully read pool: %s\n", resource_pool_name)
	return resource_pool_name, cpu_min, cpu_min_expandable, cpu_max, cpu_shares,
		mem_min, mem_min_expandable, mem_max, mem_shares, nil
}

// resourcePoolCreate_govmomi creates a resource pool using govmomi
func resourcePoolCreate(c *Config, resource_pool_name string, cpu_min int, cpu_min_expandable string,
	cpu_max int, cpu_shares string, mem_min int, mem_min_expandable string, mem_max int, mem_shares string,
	parent_pool string) (string, error) {
	log.Printf("[resourcePoolCreate] Creating pool: %s\n", resource_pool_name)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", fmt.Errorf("failed to get govmomi client: %w", err)
	}

	// Get root resource pool
	rootPool, err := getRootResourcePool(gc.Context(), gc.Finder)
	if err != nil {
		return "", fmt.Errorf("failed to get root resource pool: %w", err)
	}

	// Find parent pool
	parentPool, err := findResourcePoolByPath(gc.Context(), rootPool, parent_pool)
	if err != nil {
		return "", fmt.Errorf("failed to find parent pool: %w", err)
	}

	// Build CPU allocation spec
	cpuAllocation := buildAllocationInfo(cpu_min, cpu_min_expandable, cpu_max, cpu_shares)

	// Build Memory allocation spec
	memAllocation := buildAllocationInfo(mem_min, mem_min_expandable, mem_max, mem_shares)

	// Create resource pool spec
	spec := types.ResourceConfigSpec{
		CpuAllocation:    cpuAllocation,
		MemoryAllocation: memAllocation,
	}

	// Create the resource pool
	newPool, err := parentPool.Create(gc.Context(), resource_pool_name, spec)
	if err != nil {
		return "", fmt.Errorf("failed to create resource pool: %w", err)
	}

	poolID := newPool.Reference().Value
	log.Printf("[resourcePoolCreate_govmomi] Successfully created pool with ID: %s\n", poolID)
	return poolID, nil
}

// resourcePoolUpdate_govmomi updates a resource pool using govmomi
func resourcePoolUpdate(c *Config, pool_id string, resource_pool_name string, cpu_min int,
	cpu_min_expandable string, cpu_max int, cpu_shares string, mem_min int, mem_min_expandable string,
	mem_max int, mem_shares string) error {
	log.Printf("[resourcePoolUpdate] Updating pool ID: %s\n", pool_id)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return fmt.Errorf("failed to get govmomi client: %w", err)
	}

	// Create resource pool object from reference
	poolRef := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: pool_id,
	}
	pool := object.NewResourcePool(gc.Client.Client, poolRef)

	// Check if rename is needed
	currentName, err := getPoolNAME(c, pool_id)
	if err != nil {
		return fmt.Errorf("failed to get current pool name: %w", err)
	}

	if currentName != resource_pool_name {
		task, err := pool.Rename(gc.Context(), resource_pool_name)
		if err != nil {
			return fmt.Errorf("failed to start rename task: %w", err)
		}
		err = waitForTask(gc.Context(), task)
		if err != nil {
			return fmt.Errorf("rename task failed: %w", err)
		}
	}

	// Build CPU allocation spec
	cpuAllocation := buildAllocationInfo(cpu_min, cpu_min_expandable, cpu_max, cpu_shares)

	// Build Memory allocation spec
	memAllocation := buildAllocationInfo(mem_min, mem_min_expandable, mem_max, mem_shares)

	// Create resource pool spec
	spec := types.ResourceConfigSpec{
		CpuAllocation:    cpuAllocation,
		MemoryAllocation: memAllocation,
	}

	// Update the resource pool
	err = pool.UpdateConfig(gc.Context(), resource_pool_name, &spec)
	if err != nil {
		return fmt.Errorf("failed to update pool config: %w", err)
	}

	log.Printf("[resourcePoolUpdate_govmomi] Successfully updated pool: %s\n", resource_pool_name)
	return nil
}

// resourcePoolDelete_govmomi deletes a resource pool using govmomi
func resourcePoolDelete(c *Config, pool_id string) error {
	log.Printf("[resourcePoolDelete] Deleting pool ID: %s\n", pool_id)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return fmt.Errorf("failed to get govmomi client: %w", err)
	}

	// Create resource pool object from reference
	poolRef := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: pool_id,
	}
	pool := object.NewResourcePool(gc.Client.Client, poolRef)

	// Destroy the resource pool
	task, err := pool.Destroy(gc.Context())
	if err != nil {
		return fmt.Errorf("failed to start destroy task: %w", err)
	}

	err = waitForTask(gc.Context(), task)
	if err != nil {
		return fmt.Errorf("destroy task failed: %w", err)
	}

	log.Printf("[resourcePoolDelete_govmomi] Successfully deleted pool\n")
	return nil
}

// buildAllocationInfo is a helper function to build ResourceAllocationInfo
func buildAllocationInfo(min int, min_expandable string, max int, shares string) types.ResourceAllocationInfo {
	allocation := types.ResourceAllocationInfo{}

	// Set reservation
	if min > 0 {
		reservation := int64(min)
		allocation.Reservation = &reservation
	}

	// Set expandable reservation
	expandable := min_expandable != "false"
	allocation.ExpandableReservation = &expandable

	// Set limit
	if max > 0 {
		limit := int64(max)
		allocation.Limit = &limit
	} else {
		limit := int64(-1) // unlimited
		allocation.Limit = &limit
	}

	// Set shares
	sharesInfo := &types.SharesInfo{}
	switch strings.ToLower(shares) {
	case "low":
		sharesInfo.Level = types.SharesLevelLow
	case "high":
		sharesInfo.Level = types.SharesLevelHigh
	case "normal", "":
		sharesInfo.Level = types.SharesLevelNormal
	default:
		// Try to parse as number
		if sharesVal, err := strconv.Atoi(shares); err == nil {
			sharesInfo.Level = types.SharesLevelCustom
			sharesInfo.Shares = int32(sharesVal)
		} else {
			sharesInfo.Level = types.SharesLevelNormal
		}
	}
	allocation.Shares = sharesInfo

	return allocation
}
