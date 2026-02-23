# Phase 4: Remove SSH from Resource Pool - Research

**Researched:** 2026-02-13
**Domain:** Resource pool SSH removal in Terraform provider ESXi
**Confidence:** HIGH

## Summary

Phase 4 removes SSH code paths from the resource pool resource, following the proven pattern established in Phases 2 (portgroup) and 3 (vswitch). The resource pool has complete govmomi coverage with all CRUD operations already implemented and tested. Key differences from network resources: (1) resource pools support nested hierarchies requiring path traversal, (2) guest VMs depend on resource pools for placement, and (3) a data source exists that auto-fixes when shared functions are updated.

The govmomi implementation already handles nested pool paths via `findResourcePoolByPath` helper in `govmomi_helpers.go`, and the test suite validates create/read/delete operations work correctly. Update operations fail in the simulator (known limitation: vcsim doesn't implement Rename_Task), matching the pattern seen in portgroup/vswitch phases.

**Primary recommendation:** Apply the exact Phase 2/3 wrapper pattern to resource pool functions. All govmomi implementations exist and pass tests. The data source auto-fixes via shared function routing.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/vmware/govmomi | Latest (v0.30+) | vSphere API client | Official VMware Go library, handles all ESXi operations |
| github.com/hashicorp/terraform/helper/schema | Provider version | Terraform resource schema | Required for Terraform provider development |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/vmware/govmomi/simulator | Latest (v0.30+) | vSphere API simulator (vcsim) | All govmomi tests use this for standalone ESXi simulation |

**Installation:**
Already installed in project. No new dependencies required.

## Architecture Patterns

### Recommended Project Structure
```
esxi/
├── resource-pool_functions.go    # Core logic with wrapper + govmomi implementations
├── resource-pool_create.go       # Create operation calling govmomi
├── resource-pool_update.go       # Update operation calling govmomi
├── resource-pool_delete.go       # Delete operation calling govmomi
├── resource-pool_read.go         # Read operation (calls shared function)
├── resource-pool_import.go       # Import verification (uses shared functions)
├── resource-pool_functions_test.go  # Test coverage for govmomi operations
├── data_source_esxi_resource_pool.go  # Data source (auto-fixes via shared functions)
└── govmomi_helpers.go           # Shared helpers (getRootResourcePool, findResourcePoolByPath)
```

### Pattern 1: Thin Wrapper Functions (Phase 2/3 Established Pattern)

**What:** Replace SSH-routed functions with thin wrappers that directly call govmomi implementations

**When to use:** For all shared functions called by multiple files (getPoolID, getPoolNAME, resourcePoolRead)

**Example:**
```go
// Before (SSH routing with conditional)
func getPoolID(c *Config, resource_pool_name string) (string, error) {
    if c.useGovmomi {
        return getPoolID_govmomi(c, resource_pool_name)
    }
    // SSH code...
}

// After (thin wrapper)
func getPoolID(c *Config, resource_pool_name string) (string, error) {
    return getPoolID_govmomi(c, resource_pool_name)
}
```

### Pattern 2: Direct Govmomi Calls in CRUD Operations

**What:** Remove SSH branches and call govmomi functions directly in create/update/delete operations

**When to use:** In resource-pool_create.go, resource-pool_update.go, resource-pool_delete.go

**Example:**
```go
// Before
if c.useGovmomi {
    pool_id, err = resourcePoolCreate_govmomi(c, resource_pool_name, ...)
    if err != nil {
        d.SetId("")
        return fmt.Errorf("Failed to create pool: %s\n", err)
    }
} else {
    // SSH code...
}

// After
pool_id, err = resourcePoolCreate_govmomi(c, resource_pool_name, ...)
if err != nil {
    d.SetId("")
    return fmt.Errorf("Failed to create pool: %s\n", err)
}
```

### Pattern 3: Import Function Using Shared Functions

**What:** Rewrite import to use govmomi-based shared functions for verification

**When to use:** In resource-pool_import.go

**Example:**
```go
// resource-pool_import.go pattern (from Phase 2 portgroup)
func resourceRESOURCEPOOLImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
    c := m.(*Config)
    log.Println("[resourceRESOURCEPOOLImport]")

    results := make([]*schema.ResourceData, 1, 1)
    results[0] = d

    // Use govmomi to verify resource pool exists
    _, err := getPoolNAME(c, d.Id())
    if err != nil {
        return results, fmt.Errorf("Failed to import resource pool '%s': %s", d.Id(), err)
    }

    d.SetId(d.Id())
    return results, nil
}
```

### Pattern 4: Nested Resource Pool Path Resolution

**What:** Use existing helper functions for hierarchical pool navigation

**When to use:** Already implemented in govmomi functions, no changes needed

**Example:**
```go
// From govmomi_helpers.go (already exists)
// Handles paths like "Pool1/SubPool/ChildPool"
func findResourcePoolByPath(ctx context.Context, rootPool *object.ResourcePool, path string) (*object.ResourcePool, error) {
    if path == "/" || path == "Resources" || path == "" {
        return rootPool, nil
    }
    // Split path and traverse hierarchy...
}
```

### Anti-Patterns to Avoid

- **Don't rename _govmomi functions:** Phase 6 handles global renaming. Keep `getPoolID_govmomi`, `getPoolNAME_govmomi`, etc.
- **Don't remove useGovmomi flag:** Phase 6 removes this globally. Keep the flag in Config even though it's unused after this phase.
- **Don't modify files outside resource pool scope:** Leave config.go, esxi_remote_cmds.go, guest files, and other resources untouched.
- **Don't touch the data source directly:** `data_source_esxi_resource_pool.go` auto-fixes because it calls `getPoolID` and `resourcePoolRead` which become govmomi-only.
- **Don't expect update tests to pass:** vcsim doesn't implement Rename_Task. Test failure is expected (matches Phase 2/3 pattern).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Resource pool lookup by path | Manual path parsing and iteration | `findResourcePoolByPath` from govmomi_helpers.go | Already handles nested hierarchies, edge cases (root pool, empty path, "Resources" name) |
| Root resource pool retrieval | Direct API calls | `getRootResourcePool` from govmomi_helpers.go | Abstracts finder logic, consistent with other resources |
| Resource pool properties | Manual ManagedObjectReference construction | `object.NewResourcePool` + Properties() | Type-safe, handles error cases, validates references |
| Task waiting | Custom timeout/polling logic | `waitForTask` from govmomi_helpers.go | Already has 30-minute timeout, context cancellation |

**Key insight:** All necessary helper functions already exist and are tested. The govmomi implementation is complete and working. This phase is purely deletion of SSH code, not implementation of new functionality.

## Common Pitfalls

### Pitfall 1: Breaking Guest VM Resource Pool Dependencies
**What goes wrong:** Guest VMs call `getPoolID` during creation. If this function breaks, guest creation fails.

**Why it happens:** `guest-create.go` line 187 calls `getPoolID(c, resource_pool_name)` to place VMs in pools.

**How to avoid:** Run full test suite (`go test ./esxi/ -v`), not just resource pool tests. Verify no regressions in guest tests.

**Warning signs:** TestGuestCreateGovmomi failing, build errors in guest-create.go referencing getPoolID.

### Pitfall 2: Data Source Misconfiguration
**What goes wrong:** Assuming the data source needs explicit changes.

**Why it happens:** In Phases 2 and 3, we learned data sources auto-fix when shared functions update.

**How to avoid:** Don't modify `data_source_esxi_resource_pool.go`. It calls `getPoolID` (line 74) and `resourcePoolRead` (line 80), both of which become govmomi-only via wrapper pattern.

**Warning signs:** Manual edits to the data source file, test failures in data source tests.

### Pitfall 3: Over-Cleaning Imports
**What goes wrong:** Removing imports that are still needed by govmomi code or error handling.

**Why it happens:** SSH code uses certain imports that govmomi code also uses (fmt, log).

**How to avoid:** After removing SSH branches, run `go build ./...` to catch unused import errors. Only remove imports the compiler flags as unused.

**Warning signs:** Build errors about undefined identifiers, compiler warnings about unused imports that should have been removed.

### Pitfall 4: Nested Pool Path Edge Cases
**What goes wrong:** Assuming flat pool structure, breaking nested pool scenarios.

**Why it happens:** Not understanding that pools can be hierarchical (Pool1/SubPool).

**How to avoid:** The govmomi implementation already handles this via `findResourcePoolByPath`. Don't modify path handling logic.

**Warning signs:** Import failures for nested pools, path parsing errors in logs.

### Pitfall 5: Simulator Test Expectations
**What goes wrong:** Expecting all tests to pass, then debugging simulator limitations.

**Why it happens:** vcsim doesn't implement ResourcePool.Rename_Task.

**How to avoid:** Accept that TestResourcePoolUpdateGovmomi will fail. This is expected and documented in Phase 2/3 summaries.

**Warning signs:** Spending time debugging simulator code, attempting workarounds for Rename_Task.

## Code Examples

Verified patterns from existing codebase:

### Resource Pool Function Wrappers
```go
// Source: esxi/resource-pool_functions.go (current state with SSH, to be simplified)
// After Phase 4: Remove SSH branch, keep only govmomi call

// getPoolID wrapper
func getPoolID(c *Config, resource_pool_name string) (string, error) {
    return getPoolID_govmomi(c, resource_pool_name)
}

// getPoolNAME wrapper
func getPoolNAME(c *Config, resource_pool_id string) (string, error) {
    return getPoolNAME_govmomi(c, resource_pool_id)
}

// resourcePoolRead wrapper
func resourcePoolRead(c *Config, pool_id string) (string, int, string, int, string, int, string, int, string, error) {
    return resourcePoolRead_govmomi(c, pool_id)
}
```

### Create Operation Direct Call
```go
// Source: esxi/resource-pool_create.go (lines 102-109, remove conditional wrapper)
// After Phase 4:
pool_id, err = resourcePoolCreate_govmomi(c, resource_pool_name, cpu_min, cpu_min_expandable,
    cpu_max, cpu_shares, mem_min, mem_min_expandable, mem_max, mem_shares, parent_pool)
if err != nil {
    d.SetId("")
    return fmt.Errorf("Failed to create pool: %s\n", err)
}
```

### Update Operation Direct Call
```go
// Source: esxi/resource-pool_update.go (lines 38-49, remove SSH branch for rename, lines 101-107 remove conditional)
// Note: Phase 4 removes SSH rename (lines 38-49), keeps only govmomi path

// After Phase 4:
err = resourcePoolUpdate_govmomi(c, pool_id, resource_pool_name, cpu_min, cpu_min_expandable,
    cpu_max, cpu_shares, mem_min, mem_min_expandable, mem_max, mem_shares)
if err != nil {
    return fmt.Errorf("Failed to update pool: %s\n", err)
}
```

### Delete Operation Direct Call
```go
// Source: esxi/resource-pool_delete.go (lines 16-23, remove conditional)
// After Phase 4:
err := resourcePoolDelete_govmomi(c, pool_id)
if err != nil {
    log.Printf("[resourceRESOURCEPOOLDelete] Failed destroy resource pool id: %s\n", pool_id)
    return fmt.Errorf("Failed to delete pool: %s\n", err)
}
```

### Import Function Rewrite
```go
// Source: esxi/resource-pool_import.go (completely rewrite)
// Pattern from Phase 2 portgroup_import.go

func resourceRESOURCEPOOLImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
    c := m.(*Config)
    log.Println("[resourceRESOURCEPOOLImport]")

    results := make([]*schema.ResourceData, 1, 1)
    results[0] = d

    // Use govmomi to verify resource pool exists
    _, err := getPoolNAME(c, d.Id())
    if err != nil {
        return results, fmt.Errorf("Failed to import resource pool '%s': %s", d.Id(), err)
    }

    d.SetId(d.Id())
    return results, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SSH parsing of /etc/vmware/hostd/pools.xml | govmomi ResourcePool API | Phase 2-4 (SSH removal phases) | Eliminates XML parsing fragility, works with ESXi API changes |
| vim-cmd hostsvc/rsrc/* commands | object.ResourcePool methods | Govmomi implementation (pre-phase work) | Type-safe, handles errors properly, respects ESXi versioning |
| useGovmomi conditional routing | Direct govmomi calls | Phase 4 (this phase) | Simplifies code, removes dead SSH branches |
| SSH-based import verification | govmomi-based getPoolNAME | Phase 4 (this phase) | Consistent with other resources, no shell parsing |

**Deprecated/outdated:**
- SSH-based pool discovery via XML file parsing (lines 23-43 in resource-pool_functions.go)
- vim-cmd hostsvc/rsrc/pool_config_get for reading pool config (lines 106-189 in resource-pool_functions.go SSH branch)
- vim-cmd hostsvc/rsrc/create for pool creation (resource-pool_create.go SSH branch)
- vim-cmd hostsvc/rsrc/rename for pool rename (resource-pool_update.go SSH branch lines 38-49)
- vim-cmd hostsvc/rsrc/pool_config_set for pool updates (resource-pool_update.go SSH branch)
- vim-cmd hostsvc/rsrc/destroy for pool deletion (resource-pool_delete.go SSH branch)

## Open Questions

1. **Should we handle the root pool ("ha-root-pool") specially in import?**
   - What we know: Both SSH and govmomi implementations return "ha-root-pool" for "/" or "Resources" (lines 27-29 SSH, lines 200-202 govmomi)
   - What's unclear: Whether users import the root pool in practice
   - Recommendation: Follow existing pattern. getPoolNAME handles "ha-root-pool" -> "/" conversion (lines 61-63 SSH, lines 242-244 govmomi). Import works as-is.

2. **Do nested pool imports need special path handling?**
   - What we know: getPoolNAME_govmomi (lines 258-287) builds full paths by traversing parent hierarchy
   - What's unclear: Whether Terraform import expects pool ID or pool path as input
   - Recommendation: Import function receives pool ID (d.Id()), calls getPoolNAME to verify. This matches current implementation and portgroup pattern.

3. **Should resource-pool_update.go handle rename failures gracefully?**
   - What we know: Simulator doesn't support Rename_Task (test shows ServerFaultCode)
   - What's unclear: Whether real ESXi hosts support rename, and if failures should be non-fatal
   - Recommendation: Keep existing error handling. resourcePoolUpdate_govmomi (lines 436-451) already handles rename with proper error returns. Tests document simulator limitation.

## Sources

### Primary (HIGH confidence)
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/resource-pool_functions.go` - Current implementation with SSH and govmomi paths
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/resource-pool_create.go` - Create operation with dual-path routing
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/resource-pool_update.go` - Update operation with SSH rename and govmomi config update
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/resource-pool_delete.go` - Delete operation with dual-path routing
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/resource-pool_import.go` - Current SSH-based import
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/resource-pool_functions_test.go` - Test coverage showing govmomi works
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/data_source_esxi_resource_pool.go` - Data source that calls shared functions
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/govmomi_helpers.go` - Helper functions for resource pool operations
- `.planning/phases/02-remove-ssh-from-portgroup/02-01-PLAN.md` - Established SSH removal pattern
- `.planning/phases/03-remove-ssh-from-vswitch/03-01-PLAN.md` - Validated SSH removal pattern on second resource
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/guest-create.go` - Guest VM dependency on getPoolID
- `/home/cars/src/github/cars/terraform-provider-esxi/esxi/guest-read.go` - Guest VM dependency on getPoolNAME

### Secondary (MEDIUM confidence)
- Test output from `go test ./esxi/ -v -run TestResourcePool` - Confirms govmomi implementation passes create/read/delete tests, update fails due to simulator limitation (expected)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use, no new dependencies
- Architecture: HIGH - Pattern proven in Phases 2 and 3, all govmomi code exists and tested
- Pitfalls: HIGH - Based on actual Phase 2/3 experience, codebase analysis shows dependencies

**Research date:** 2026-02-13
**Valid until:** 2026-03-15 (30 days for stable codebase with established patterns)
