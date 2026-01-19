package esxi

import (
	"bufio"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

//  Check if Pool exists (by name )and return it's Pool ID.
func getPoolID(c *Config, resource_pool_name string) (string, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return getPoolID_govmomi(c, resource_pool_name)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Printf("[getPoolID]\n")

	if resource_pool_name == "/" || resource_pool_name == "Resources" {
		return "ha-root-pool", nil
	}

	result := strings.Split(resource_pool_name, "/")
	resource_pool_name = result[len(result)-1]

	r := strings.NewReplacer("objID>", "", "</objID", "")
	remote_cmd := fmt.Sprintf("grep -A1 '<name>%s</name>' /etc/vmware/hostd/pools.xml | grep -m 1 -o objID.*objID", resource_pool_name)
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get existing resource pool id")
	if err == nil {
		stdout = r.Replace(stdout)
		return stdout, err
	} else {
		log.Printf("[getPoolID] Failed get existing resource pool id: %s\n", stdout)
		return "", err
	}
}

//  Check if Pool exists (by id)and return it's Pool name.
func getPoolNAME(c *Config, resource_pool_id string) (string, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return getPoolNAME_govmomi(c, resource_pool_id)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Printf("[getPoolNAME]\n")

	var ResourcePoolName, fullResourcePoolName string

	fullResourcePoolName = ""

	if resource_pool_id == "ha-root-pool" {
		return "/", nil
	}

	// Get full Resource Pool Path
	remote_cmd := fmt.Sprintf("grep -A1 '<objID>%s</objID>' /etc/vmware/hostd/pools.xml | grep '<path>'", resource_pool_id)
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get resource pool path")
	if err != nil {
		log.Printf("[getPoolNAME] Failed get resource pool PATH: %s\n", stdout)
		return "", fmt.Errorf("Failed to get pool path: %s\n", err)
	}

	re := regexp.MustCompile(`[/<>\n]`)
	result := re.Split(stdout, -1)

	for i := range result {

		ResourcePoolName = ""
		if result[i] != "path" && result[i] != "host" && result[i] != "user" && result[i] != "" {

			r := strings.NewReplacer("name>", "", "</name", "")
			remote_cmd := fmt.Sprintf("grep -B1 '<objID>%s</objID>' /etc/vmware/hostd/pools.xml | grep -o name.*name", result[i])
			stdout, _ := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get resource pool name")
			ResourcePoolName = r.Replace(stdout)

			if ResourcePoolName != "" {
				if result[i] == resource_pool_id {
					fullResourcePoolName = fullResourcePoolName + ResourcePoolName
				} else {
					fullResourcePoolName = fullResourcePoolName + ResourcePoolName + "/"
				}
			}
		}
	}

	return fullResourcePoolName, nil
}

func resourcePoolRead(c *Config, pool_id string) (string, int, string, int, string, int, string, int, string, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return resourcePoolRead_govmomi(c, pool_id)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Println("[resourcePoolRead]")

	var remote_cmd, stdout, cpu_shares, mem_shares string
	var cpu_min, cpu_max, mem_min, mem_max, tmpvar int
	var cpu_min_expandable, mem_min_expandable string
	var err error

	remote_cmd = fmt.Sprintf("vim-cmd hostsvc/rsrc/pool_config_get %s", pool_id)
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "resource pool_config_get")

	if strings.Contains(stdout, "deleted") == true {
		log.Printf("[resourcePoolRead] Already deleted: %s\n", err)
		return "", 0, "", 0, "", 0, "", 0, "", nil
	}
	if err != nil {
		log.Printf("[resourcePoolRead] Failed to get %s: %s\n", "resource pool_config_get", err)
		return "", 0, "", 0, "", 0, "", 0, "", fmt.Errorf("Failed to get pool config: %s\n", err)
	}

	is_cpu_flag := true

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		switch {
		case strings.Contains(scanner.Text(), "memoryAllocation = "):
			is_cpu_flag = false

		case strings.Contains(scanner.Text(), "reservation = "):
			r, _ := regexp.Compile("[0-9]+")
			if is_cpu_flag == true {
				cpu_min, _ = strconv.Atoi(r.FindString(scanner.Text()))
			} else {
				mem_min, _ = strconv.Atoi(r.FindString(scanner.Text()))
			}

		case strings.Contains(scanner.Text(), "expandableReservation = "):
			r, _ := regexp.Compile("(true|false)")
			if is_cpu_flag == true {
				cpu_min_expandable = r.FindString(scanner.Text())
			} else {
				mem_min_expandable = r.FindString(scanner.Text())
			}

		case strings.Contains(scanner.Text(), "limit = "):
			r, _ := regexp.Compile("-?[0-9]+")
			tmpvar, _ = strconv.Atoi(r.FindString(scanner.Text()))
			if tmpvar < 0 {
				tmpvar = 0
			}
			if is_cpu_flag == true {
				cpu_max = tmpvar
			} else {
				mem_max = tmpvar
			}

		case strings.Contains(scanner.Text(), "shares = "):
			r, _ := regexp.Compile("[0-9]+")
			if is_cpu_flag == true {
				cpu_shares = r.FindString(scanner.Text())
			} else {
				mem_shares = r.FindString(scanner.Text())
			}

		case strings.Contains(scanner.Text(), "level = "):
			r, _ := regexp.Compile("(low|high|normal)")
			if r.FindString(scanner.Text()) != "" {
				if is_cpu_flag == true {
					cpu_shares = r.FindString(scanner.Text())
				} else {
					mem_shares = r.FindString(scanner.Text())
				}
			}
		}
	}

	resource_pool_name, err := getPoolNAME(c, pool_id)
	if err != nil {
		log.Printf("[resourcePoolRead] Failed to get Resource Pool name: %s\n", err)
		return "", 0, "", 0, "", 0, "", 0, "", fmt.Errorf("Failed to get pool name: %s\n", err)
	}

	return resource_pool_name, cpu_min, cpu_min_expandable, cpu_max, cpu_shares,
		mem_min, mem_min_expandable, mem_max, mem_shares, nil
}

// ============================================================================
// Govmomi-based Resource Pool Operations
// ============================================================================

// getPoolID_govmomi gets resource pool ID by name using govmomi
func getPoolID_govmomi(c *Config, resource_pool_name string) (string, error) {
	log.Printf("[getPoolID_govmomi] Getting pool ID for: %s\n", resource_pool_name)

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

// getPoolNAME_govmomi gets resource pool name by ID using govmomi
func getPoolNAME_govmomi(c *Config, resource_pool_id string) (string, error) {
	log.Printf("[getPoolNAME_govmomi] Getting pool name for ID: %s\n", resource_pool_id)

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

// resourcePoolRead_govmomi reads resource pool configuration using govmomi
func resourcePoolRead_govmomi(c *Config, pool_id string) (string, int, string, int, string, int, string, int, string, error) {
	log.Printf("[resourcePoolRead_govmomi] Reading pool ID: %s\n", pool_id)

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
	resource_pool_name, err := getPoolNAME_govmomi(c, pool_id)
	if err != nil {
		return "", 0, "", 0, "", 0, "", 0, "", fmt.Errorf("failed to get pool name: %w", err)
	}

	log.Printf("[resourcePoolRead_govmomi] Successfully read pool: %s\n", resource_pool_name)
	return resource_pool_name, cpu_min, cpu_min_expandable, cpu_max, cpu_shares,
		mem_min, mem_min_expandable, mem_max, mem_shares, nil
}

// resourcePoolCreate_govmomi creates a resource pool using govmomi
func resourcePoolCreate_govmomi(c *Config, resource_pool_name string, cpu_min int, cpu_min_expandable string,
	cpu_max int, cpu_shares string, mem_min int, mem_min_expandable string, mem_max int, mem_shares string,
	parent_pool string) (string, error) {
	log.Printf("[resourcePoolCreate_govmomi] Creating pool: %s\n", resource_pool_name)

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
func resourcePoolUpdate_govmomi(c *Config, pool_id string, resource_pool_name string, cpu_min int,
	cpu_min_expandable string, cpu_max int, cpu_shares string, mem_min int, mem_min_expandable string,
	mem_max int, mem_shares string) error {
	log.Printf("[resourcePoolUpdate_govmomi] Updating pool ID: %s\n", pool_id)

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
	currentName, err := getPoolNAME_govmomi(c, pool_id)
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
func resourcePoolDelete_govmomi(c *Config, pool_id string) error {
	log.Printf("[resourcePoolDelete_govmomi] Deleting pool ID: %s\n", pool_id)

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
