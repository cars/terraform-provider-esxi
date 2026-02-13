# Roadmap: Terraform Provider ESXi — Build Fix & SSH Removal

**Created:** 2026-02-12
**Depth:** Standard (6 phases)
**Coverage:** 22/22 requirements mapped

## Overview

Transform the provider from dual-path (SSH + govmomi) to govmomi-first by fixing build errors, then systematically removing SSH code from resources with complete govmomi coverage. Guest VM operations retain SSH where no govmomi alternative exists. Each phase delivers a verifiable capability and maintains test coverage throughout.

## Phases

### Phase 1: Fix Build Errors

**Goal:** Provider compiles cleanly and all existing tests pass

**Dependencies:** None (foundation phase)

**Requirements:** BUILD-01, BUILD-02, BUILD-03

**Plans:** 1 plan

Plans:
- [ ] 01-01-PLAN.md — Fix type mismatches and implement missing govmomi function

**Success Criteria:**
1. `go build ./...` succeeds without compilation errors
2. `esxi_host` data source implements govmomi read function (dataSourceEsxiHostReadGovmomi exists)
3. All existing tests pass (`go test ./esxi/ -v` returns 0 failures)
4. Git commit created documenting build fix changes

**Reasoning:** Cannot proceed with SSH removal until codebase compiles. Establishes baseline test coverage and validates session management patterns work correctly. Research flags this as blocking issue requiring resolution before any other work.

---

### Phase 2: Remove SSH from Portgroup

**Goal:** Portgroup resource operates entirely via govmomi API

**Dependencies:** Phase 1 (requires working build and tests)

**Requirements:** PORT-01, PORT-02, PORT-03, PORT-04

**Plans:** 1 plan

Plans:
- [ ] 02-01-PLAN.md — Remove SSH branches from portgroup CRUD and rewrite import

**Success Criteria:**
1. portgroup_functions.go contains no SSH code paths (portgroupCreate, portgroupRead, portgroupSecurityPolicyRead use govmomi only)
2. portgroup_import.go successfully imports portgroups using govmomi
3. All portgroup tests pass with SSH code removed
4. Git commit created documenting portgroup SSH removal

**Reasoning:** Smallest resource with complete govmomi coverage. Establishes removal pattern for subsequent phases. Low complexity, low risk proof of concept. Research identifies this as pattern-establishing phase.

---

### Phase 3: Remove SSH from vSwitch

**Goal:** vSwitch resource operates entirely via govmomi API

**Dependencies:** Phase 2 (follows proven removal pattern, portgroup depends on vswitch)

**Requirements:** VSW-01, VSW-02, VSW-03, VSW-04

**Plans:** 1 plan

Plans:
- [ ] 03-01-PLAN.md — Remove SSH branches from vSwitch CRUD and rewrite import

**Success Criteria:**
1. vswitch_functions.go contains no SSH code paths (vswitchUpdate, vswitchRead use govmomi only)
2. vswitch_import.go successfully imports vswitches using govmomi
3. All vswitch tests pass with SSH code removed
4. Portgroup tests still pass (validates network dependency handling)
5. Git commit created documenting vswitch SSH removal

**Reasoning:** Completes networking resource cleanup. Validates removal pattern works across multiple resources. Tests confirm dependency handling between vswitch and portgroup works correctly.

---

### Phase 4: Remove SSH from Resource Pool

**Goal:** Resource pool resource operates entirely via govmomi API

**Dependencies:** Phase 1 (requires working build and tests)

**Requirements:** RPOOL-01, RPOOL-02, RPOOL-03

**Success Criteria:**
1. resource-pool_functions.go contains no SSH code paths (getPoolID, getPoolNAME, resourcePoolRead use govmomi only)
2. Nested resource pool scenarios work correctly (path resolution via govmomi)
3. All resource pool tests pass with SSH code removed
4. Git commit created documenting resource pool SSH removal

**Reasoning:** First compute resource cleanup. Must complete before virtual disk (guest VMs depend on both). Research identifies ManagedObjectReference handling and path resolution as key validation points.

---

### Phase 5: Remove SSH from Virtual Disk

**Goal:** Virtual disk resource operates entirely via govmomi API

**Dependencies:** Phase 1 (requires working build and tests)

**Requirements:** VDISK-01, VDISK-02, VDISK-03, VDISK-04

**Success Criteria:**
1. virtual-disk_functions.go contains no SSH code paths (diskStoreValidate, virtualDiskCREATE, virtualDiskREAD, growVirtualDisk use govmomi only)
2. Virtual disk delete operation works via govmomi (virtualDiskDelete_govmomi implemented if missing)
3. Disk path format standardized ([datastore] path/disk.vmdk) across create/read/grow/delete
4. data_source_esxi_virtual_disk operates without SSH fallback
5. All virtual disk tests pass with SSH code removed
6. Git commit created documenting virtual disk SSH removal

**Reasoning:** Completes SSH removal from all fully-covered resources. Most complex removal due to datastore browser API usage and path format standardization. Research flags disk path format inconsistency as key pitfall to address.

---

### Phase 6: Infrastructure Cleanup

**Goal:** Provider architecture simplified with SSH retained only for guest operations

**Dependencies:** Phases 2, 3, 4, 5 (all SSH removals complete)

**Requirements:** INFRA-01, INFRA-02, INFRA-03, INFRA-04

**Success Criteria:**
1. useGovmomi flag removed from portgroup, vswitch, resource_pool, virtual_disk resources (call govmomi directly)
2. Unused SSH imports removed from modified files
3. esxi_remote_cmds.go, ConnectionStruct, and SSH code remain available for guest operations
4. Provider builds cleanly (`go build ./...` succeeds)
5. All tests pass after cleanup (`go test ./esxi/ -v` returns 0 failures)
6. Git commit created documenting infrastructure cleanup

**Reasoning:** Simplifies architecture after removals complete. Eliminates dual-path routing confusion. Documents intentional SSH retention for guest create/delete operations (not technical debt). Research identifies this as critical clarity phase.

---

## Progress

| Phase | Status | Requirements | Completion |
|-------|--------|--------------|------------|
| 1 - Fix Build | ✓ Complete (2026-02-13) | BUILD-01, BUILD-02, BUILD-03 | 100% |
| 2 - Portgroup SSH Removal | ✓ Complete (2026-02-13) | PORT-01, PORT-02, PORT-03, PORT-04 | 100% |
| 3 - vSwitch SSH Removal | Pending | VSW-01, VSW-02, VSW-03, VSW-04 | 0% |
| 4 - Resource Pool SSH Removal | Pending | RPOOL-01, RPOOL-02, RPOOL-03 | 0% |
| 5 - Virtual Disk SSH Removal | Pending | VDISK-01, VDISK-02, VDISK-03, VDISK-04 | 0% |
| 6 - Infrastructure Cleanup | Pending | INFRA-01, INFRA-02, INFRA-03, INFRA-04 | 0% |

---

## Coverage Validation

**Total v1 Requirements:** 22

**Mapped:**
- Phase 1: 3 requirements (BUILD-01, BUILD-02, BUILD-03)
- Phase 2: 4 requirements (PORT-01, PORT-02, PORT-03, PORT-04)
- Phase 3: 4 requirements (VSW-01, VSW-02, VSW-03, VSW-04)
- Phase 4: 3 requirements (RPOOL-01, RPOOL-02, RPOOL-03)
- Phase 5: 4 requirements (VDISK-01, VDISK-02, VDISK-03, VDISK-04)
- Phase 6: 4 requirements (INFRA-01, INFRA-02, INFRA-03, INFRA-04)

**Unmapped:** 0

**Coverage:** 22/22 (100%)

---

## Research Integration

Research summary identified 7 suggested phases. Roadmap uses 6 phases by combining research insights:

- Research Phase 1 (Fix Build) → Roadmap Phase 1
- Research Phase 2 (Portgroup) → Roadmap Phase 2
- Research Phase 3 (vSwitch) → Roadmap Phase 3
- Research Phase 4 (Resource Pool) → Roadmap Phase 4
- Research Phase 5 (Virtual Disk) → Roadmap Phase 5
- Research Phase 6 (Infrastructure Cleanup) → Roadmap Phase 6
- Research Phase 7 (Test Hardening) → Not in v1 requirements (potential v2 enhancement)

Research flags carried forward:
- Phase 1: Need to decide esxi_host data source implementation strategy
- Phase 5: Verify virtualDiskDelete_govmomi exists before removal
- Phase 2-3: Verify import functionality with SSH-created resources

---

*Last updated: 2026-02-13 (Phase 2 complete)*
