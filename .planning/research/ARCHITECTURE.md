# Architecture Research: SSH Removal Strategy

**Domain:** Terraform provider dual-path to govmomi-only migration
**Researched:** 2026-02-12
**Overall confidence:** HIGH (based on codebase analysis)

## Current Architecture

```
Provider (provider.go)
  ├── Config (config.go) — holds credentials + cached GovmomiClient
  │     ├── SSH: ConnectionStruct → runRemoteSshCommand()
  │     └── govmomi: GovmomiClient → govmomi API calls
  │
  ├── Resources (5)
  │     ├── guest: schema → CRUD files → guest_functions.go (PARTIAL govmomi)
  │     ├── portgroup: schema → CRUD files → portgroup_functions.go (FULL govmomi)
  │     ├── vswitch: schema → CRUD files → vswitch_functions.go (FULL govmomi)
  │     ├── resource_pool: schema → CRUD files → resource-pool_functions.go (FULL govmomi)
  │     └── virtual_disk: schema → CRUD files → virtual-disk_functions.go (FULL govmomi)
  │
  └── Data Sources (6)
        ├── esxi_guest (govmomi read exists)
        ├── esxi_host (BROKEN — missing govmomi function)
        ├── esxi_portgroup (govmomi)
        ├── esxi_vswitch (govmomi)
        ├── esxi_resource_pool (govmomi)
        └── esxi_virtual_disk (govmomi)
```

## Target Architecture (After Cleanup)

```
Provider (provider.go)
  ├── Config (config.go) — credentials + GovmomiClient + SSH for guest ops
  │     ├── SSH: ConnectionStruct → runRemoteSshCommand() [RETAINED for guest]
  │     └── govmomi: GovmomiClient → govmomi API calls [PRIMARY]
  │
  ├── Resources
  │     ├── guest: schema → CRUD → guest_functions.go (SSH create/delete, govmomi read/power)
  │     ├── portgroup: schema → CRUD → portgroup_functions.go (govmomi ONLY)
  │     ├── vswitch: schema → CRUD → vswitch_functions.go (govmomi ONLY)
  │     ├── resource_pool: schema → CRUD → resource-pool_functions.go (govmomi ONLY)
  │     └── virtual_disk: schema → CRUD → virtual-disk_functions.go (govmomi ONLY)
  │
  └── Data Sources — all govmomi
```

## Files to Modify vs Remove

### Files to modify (remove SSH branches, keep govmomi):

| File | Change | Complexity |
|------|--------|------------|
| portgroup_functions.go | Remove SSH branches from 3 functions | LOW |
| portgroup_create.go | Remove SSH branch | LOW |
| portgroup_update.go | Remove SSH branches | LOW |
| portgroup_delete.go | Remove SSH branch | LOW |
| portgroup_import.go | Rewrite to use govmomi | LOW |
| vswitch_functions.go | Remove SSH branches from 2 functions | LOW |
| vswitch_create.go | Remove SSH branch | LOW |
| vswitch_delete.go | Remove SSH branch | LOW |
| vswitch_import.go | Rewrite to use govmomi | LOW |
| resource-pool_functions.go | Remove SSH branches from 3 functions | LOW |
| resource-pool_create.go | Remove SSH branch | LOW |
| resource-pool_update.go | Remove SSH branches | LOW |
| resource-pool_delete.go | Remove SSH branch | LOW |
| virtual-disk_functions.go | Remove SSH branches from 4 functions | MEDIUM |
| virtual-disk_delete.go | Implement govmomi or remove SSH branch | MEDIUM |
| data_source_esxi_host.go | Fix build errors (type mismatches + missing function) | MEDIUM |
| data_source_esxi_virtual_disk.go | Remove SSH branch | LOW |
| config.go | Remove `useGovmomi` flag (make govmomi default) | LOW |

### Files to leave unchanged:

| File | Reason |
|------|--------|
| guest_functions.go | Still needs SSH for several operations |
| guest-create.go | No govmomi CREATE implementation |
| guest-delete.go | No govmomi DELETE implementation |
| guest-read.go | Has govmomi branch but SSH still needed as fallback |
| guest_update.go | May reference SSH operations |
| esxi_remote_cmds.go | Still used by guest operations |
| connection_info.go | SSH connection struct still needed |
| data_source_esxi_guest.go | Has govmomi branch, may keep both |

## Safe Removal Order

**Phase 1: Fix Build (no SSH removal yet)**
1. Fix data_source_esxi_host.go build errors
2. Verify all tests pass

**Phase 2: Remove SSH from fully-covered resources (simplest first)**
1. Portgroup (smallest, full govmomi, good test coverage)
2. vSwitch (similar to portgroup, full govmomi)
3. Resource Pool (full govmomi, test coverage exists)
4. Virtual Disk (full govmomi, slightly more complex)

**Phase 3: Clean up shared infrastructure**
1. Remove `useGovmomi` flag from Config (always use govmomi)
2. Clean up import functions
3. Update data sources
4. Remove unused SSH code from functions files

**Phase 4: Fix tests**
1. Update tests to remove SSH path testing where removed
2. Verify all govmomi tests pass
3. Add missing test coverage

## Key Risk: Config.useGovmomi Flag

Currently the routing flag `c.useGovmomi` exists in Config. After removing SSH from 4 resources:
- The flag still matters for guest operations (SSH create/delete)
- Consider: make govmomi the default, SSH fallback for guest-specific operations
- OR: Remove the flag entirely, call SSH directly in guest-specific code (no routing)

**Recommended approach:** Remove the `if c.useGovmomi` routing pattern. For resources with full govmomi, call govmomi directly. For guest operations that need SSH, call SSH directly without the flag check.

## Component Boundaries

**Keep:**
- govmomi_client.go — core connection management
- govmomi_helpers.go — shared VM lookup, power ops, task waiting
- esxi_remote_cmds.go — still needed for guest SSH operations
- All _govmomi functions in *_functions.go files

**Remove (from modified files):**
- SSH branches in `if c.useGovmomi { ... } else { SSH code }` blocks
- SSH-only helper code no longer called by any function
- Unused imports after SSH code removal

**Do NOT remove:**
- esxi_remote_cmds.go (guest still needs it)
- SSH imports in guest files
- ConnectionStruct (still used)

---

*Architecture research: 2026-02-12*
