# Feature Research: Terraform Provider ESXi SSH/govmomi Coverage

**Domain:** ESXi Terraform provider operations — SSH vs govmomi coverage
**Researched:** 2026-02-12
**Overall confidence:** HIGH (based on codebase grep analysis)

## Coverage Matrix

### Portgroup — FULL govmomi coverage (can remove SSH)

| Operation | SSH | govmomi | Can Remove SSH? |
|-----------|-----|---------|-----------------|
| Create | portgroup_create.go:33 | portgroupCreate_govmomi | YES |
| Read | portgroup_functions.go:38,70 | portgroupRead_govmomi, portgroupSecurityPolicyRead_govmomi | YES |
| Update | portgroup_update.go:50,71 | portgroupUpdate_govmomi | YES |
| Delete | portgroup_delete.go:32 | portgroupDelete_govmomi | YES |
| Import | portgroup_import.go:24 | Need to verify | CHECK |

**Complexity:** LOW — All CRUD operations have govmomi alternatives.

### vSwitch — FULL govmomi coverage (can remove SSH)

| Operation | SSH | govmomi | Can Remove SSH? |
|-----------|-----|---------|-----------------|
| Create | vswitch_create.go:79 | vswitchCreate_govmomi | YES |
| Read | vswitch_functions.go:116,156 | vswitchRead_govmomi | YES |
| Update | vswitch_functions.go:34,43,50,71,88 | vswitchUpdate_govmomi | YES |
| Delete | vswitch_delete.go:28 | vswitchDelete_govmomi | YES |
| Import | vswitch_import.go:24 | Need to verify | CHECK |

**Complexity:** LOW — All CRUD operations have govmomi alternatives.

### Resource Pool — FULL govmomi coverage (can remove SSH)

| Operation | SSH | govmomi | Can Remove SSH? |
|-----------|-----|---------|-----------------|
| Create | resource-pool_create.go:122 | resourcePoolCreate_govmomi | YES |
| Read | resource-pool_functions.go:115 | resourcePoolRead_govmomi | YES |
| Update | resource-pool_update.go:45,114 | resourcePoolUpdate_govmomi | YES |
| Delete | resource-pool_delete.go:30 | resourcePoolDelete_govmomi | YES |
| GetID | resource-pool_functions.go:36 | getPoolID_govmomi | YES |
| GetName | resource-pool_functions.go:67,83 | getPoolNAME_govmomi | YES |

**Complexity:** LOW — All CRUD operations have govmomi alternatives.

### Virtual Disk — FULL govmomi coverage (can remove SSH)

| Operation | SSH | govmomi | Can Remove SSH? |
|-----------|-----|---------|-----------------|
| Create | virtual-disk_functions.go:87-112 | virtualDiskCREATE_govmomi | YES |
| Read | virtual-disk_functions.go:188-217 | virtualDiskREAD_govmomi | YES |
| Grow | virtual-disk_functions.go:144 | growVirtualDisk_govmomi | YES |
| Validate | virtual-disk_functions.go:34-45 | diskStoreValidate_govmomi | YES |
| Delete | virtual-disk_delete.go:25,37,42 | Need to verify | CHECK |

**Complexity:** LOW-MEDIUM — Most operations covered. Disk delete needs verification.

### Guest (VM) — PARTIAL govmomi coverage (keep SSH for missing operations)

| Operation | SSH | govmomi | Can Remove SSH? |
|-----------|-----|---------|-----------------|
| GetVMID | guest_functions.go:28 | guestGetVMID_govmomi | YES |
| ValidateVMID | guest_functions.go:54 | guestValidateVMID_govmomi | YES |
| PowerOn | guest_functions.go:420 | guestPowerOn_govmomi | YES |
| PowerOff | guest_functions.go:450,462,469 | guestPowerOff_govmomi | YES |
| PowerGetState | guest_functions.go:485 | guestPowerGetState_govmomi | YES |
| GetIpAddress | guest_functions.go:526,536,547 | guestGetIpAddress_govmomi | YES |
| Read | guest-read.go:111-162 | guestREAD_govmomi | YES |
| DeviceInfo | — | guestReadDevices_govmomi | govmomi-only |
| **CREATE** | guest-create.go (78-298) | **MISSING** | **NO — keep SSH** |
| **DELETE** | guest-delete.go:35 | **MISSING** | **NO — keep SSH** |
| GetBootDisk | guest_functions.go:72 | **MISSING** | **NO — keep SSH** |
| GetDstVmx | guest_functions.go:90,96 | **MISSING** | **NO — keep SSH** |
| ReadVMX | guest_functions.go:111 | **MISSING** | **NO — keep SSH** |
| Reload | guest_functions.go:400 | **MISSING** | **NO — keep SSH** |

**Complexity:** HIGH — CREATE and DELETE are the most critical operations and have no govmomi alternative. These are large functions with OVF handling, cloud-init, SCP, etc.

### Data Sources

| Data Source | SSH | govmomi | Can Remove SSH? |
|-------------|-----|---------|-----------------|
| esxi_guest | Has useGovmomi branch | guestREAD_govmomi | YES (for read path) |
| esxi_host | Has useGovmomi branch | **MISSING: dataSourceEsxiHostReadGovmomi** | **NO — needs implementation** |
| esxi_portgroup | Uses govmomi directly | portgroupRead_govmomi | Already govmomi |
| esxi_vswitch | Uses govmomi directly | vswitchRead_govmomi | Already govmomi |
| esxi_resource_pool | Uses govmomi directly | resourcePoolRead_govmomi | Already govmomi |
| esxi_virtual_disk | Has useGovmomi branch | virtualDiskREAD_govmomi | YES |

### Config/Provider

| Operation | SSH | govmomi | Can Remove SSH? |
|-----------|-----|---------|-----------------|
| Connectivity test | config.go:57 | No equivalent | **NO — keep SSH** |
| Create home dir | config.go:62 | No equivalent | **NO — keep SSH** |

## Summary: What Can Be Removed

**SSH code removable (4 resources):**
- Portgroup: all SSH code
- vSwitch: all SSH code
- Resource Pool: all SSH code
- Virtual Disk: all SSH code (verify delete)

**SSH code must stay:**
- Guest CREATE (no govmomi)
- Guest DELETE (no govmomi)
- Guest utility functions (boot disk, VMX, reload)
- Data source esxi_host SSH helpers (govmomi function missing — needs implementation OR keep SSH)
- Config SSH connectivity test
- esxi_remote_cmds.go (still used by above)

**Data source fixes needed:**
- data_source_esxi_host.go: missing `dataSourceEsxiHostReadGovmomi` function (BUILD BLOCKER)
- Type mismatches in SSH helper functions

## Feature Dependencies

```
portgroup → vswitch (portgroup needs vswitch to exist)
guest → resource_pool (guest placed in pool)
guest → virtual_disk (guest uses disks)
guest → portgroup (guest connects to networks)
```

---

*Feature research: 2026-02-12*
