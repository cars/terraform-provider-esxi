# Requirements: Terraform Provider ESXi — Build Fix & SSH Removal

**Defined:** 2026-02-12
**Core Value:** The provider must compile cleanly and all tests must pass

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Build Fix

- [ ] **BUILD-01**: Provider compiles without errors (`go build ./...` succeeds)
- [ ] **BUILD-02**: Implement `dataSourceEsxiHostReadGovmomi` function for esxi_host data source
- [ ] **BUILD-03**: All existing tests pass after build fix (`go test ./esxi/ -v`)

### Portgroup SSH Removal

- [ ] **PORT-01**: Remove SSH code paths from portgroup_functions.go (portgroupCreate, portgroupRead, portgroupSecurityPolicyRead)
- [ ] **PORT-02**: Remove SSH code paths from portgroup_create.go, portgroup_update.go, portgroup_delete.go
- [ ] **PORT-03**: Rewrite portgroup_import.go to use govmomi instead of SSH
- [ ] **PORT-04**: Portgroup tests pass with govmomi-only implementation

### vSwitch SSH Removal

- [ ] **VSW-01**: Remove SSH code paths from vswitch_functions.go (vswitchUpdate, vswitchRead)
- [ ] **VSW-02**: Remove SSH code paths from vswitch_create.go, vswitch_delete.go
- [ ] **VSW-03**: Rewrite vswitch_import.go to use govmomi instead of SSH
- [ ] **VSW-04**: vSwitch tests pass with govmomi-only implementation

### Resource Pool SSH Removal

- [ ] **RPOOL-01**: Remove SSH code paths from resource-pool_functions.go (getPoolID, getPoolNAME, resourcePoolRead)
- [ ] **RPOOL-02**: Remove SSH code paths from resource-pool_create.go, resource-pool_update.go, resource-pool_delete.go
- [ ] **RPOOL-03**: Resource pool tests pass with govmomi-only implementation

### Virtual Disk SSH Removal

- [ ] **VDISK-01**: Remove SSH code paths from virtual-disk_functions.go (diskStoreValidate, virtualDiskCREATE, virtualDiskREAD, growVirtualDisk)
- [ ] **VDISK-02**: Remove SSH code paths from virtual-disk_delete.go (implement govmomi delete if missing)
- [ ] **VDISK-03**: Remove SSH fallback from data_source_esxi_virtual_disk.go
- [ ] **VDISK-04**: Virtual disk tests pass with govmomi-only implementation

### Infrastructure Cleanup

- [ ] **INFRA-01**: Remove `useGovmomi` flag routing from cleaned resources (call govmomi directly)
- [ ] **INFRA-02**: Remove unused SSH imports from modified files
- [ ] **INFRA-03**: Keep esxi_remote_cmds.go, ConnectionStruct, and SSH code used by guest operations
- [ ] **INFRA-04**: Provider builds and all tests pass after full cleanup

## v2 Requirements

Deferred to future milestone. Not in current roadmap.

### Guest VM govmomi Migration

- **GUEST-01**: Implement guestCREATE_govmomi for VM creation without SSH
- **GUEST-02**: Implement guestDELETE_govmomi for VM deletion without SSH
- **GUEST-03**: Implement govmomi equivalents for guest utility functions (boot disk, VMX read, reload)
- **GUEST-04**: Remove all SSH code from guest operations

### Provider Modernization

- **MOD-01**: Upgrade from Terraform SDK v0.12.17 to terraform-plugin-framework
- **MOD-02**: Upgrade govmomi to latest version
- **MOD-03**: Remove x/crypto and tmc/scp dependencies after full SSH removal

## Out of Scope

| Feature | Reason |
|---------|--------|
| Guest CREATE/DELETE govmomi implementation | Complex operations (OVF, cloud-init, SCP), defer to v2 |
| Terraform SDK upgrade | Massive rewrite, separate project |
| govmomi version upgrade | Unnecessary risk during cleanup |
| Adding new resources or data sources | Focus on fixing and cleaning existing code |
| Refactoring existing govmomi code | Leave working implementations as-is |
| Performance optimization | Not the goal of this milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUILD-01 | Phase 1 | Pending |
| BUILD-02 | Phase 1 | Pending |
| BUILD-03 | Phase 1 | Pending |
| PORT-01 | Phase 2 | Pending |
| PORT-02 | Phase 2 | Pending |
| PORT-03 | Phase 2 | Pending |
| PORT-04 | Phase 2 | Pending |
| VSW-01 | Phase 3 | Pending |
| VSW-02 | Phase 3 | Pending |
| VSW-03 | Phase 3 | Pending |
| VSW-04 | Phase 3 | Pending |
| RPOOL-01 | Phase 4 | Pending |
| RPOOL-02 | Phase 4 | Pending |
| RPOOL-03 | Phase 4 | Pending |
| VDISK-01 | Phase 5 | Pending |
| VDISK-02 | Phase 5 | Pending |
| VDISK-03 | Phase 5 | Pending |
| VDISK-04 | Phase 5 | Pending |
| INFRA-01 | Phase 6 | Pending |
| INFRA-02 | Phase 6 | Pending |
| INFRA-03 | Phase 6 | Pending |
| INFRA-04 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-12*
*Last updated: 2026-02-12 after initial definition*
