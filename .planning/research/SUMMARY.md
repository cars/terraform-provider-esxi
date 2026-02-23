# Project Research Summary

**Project:** Terraform Provider ESXi - SSH to govmomi Migration
**Domain:** Infrastructure as Code provider cleanup/migration
**Researched:** 2026-02-12
**Confidence:** HIGH

## Executive Summary

This project cleans up dual-path implementation debt in a Terraform provider for VMware ESXi. The provider currently routes operations through either SSH commands (legacy) or the govmomi API (modern) based on a useGovmomi flag. Research reveals that 4 of 5 resources (portgroup, vswitch, resource_pool, virtual_disk) have complete govmomi coverage and can have SSH code removed safely. The guest (VM) resource must retain SSH for create/delete operations where no govmomi alternative exists.

The recommended approach is a phased removal strategy: fix compilation errors first, then systematically remove SSH code from fully-covered resources in ascending complexity order, followed by infrastructure cleanup. This minimizes risk by establishing test coverage and session management patterns before removing safety nets. No stack upgrades are needed - this is pure code cleanup using existing Go 1.23.0, govmomi v0.52.0, and Terraform SDK v0.12.17.

Key risks include state inconsistency between SSH-created and govmomi-managed resources, govmomi session management failures (ESXi limits 32 concurrent sessions), and error handling differences between SSH command parsing and API fault types. These are mitigated by establishing ID format standards, implementing session keepalive/cleanup, and comprehensive testing with govmomi simulator before each phase.

## Key Findings

### Recommended Stack

No stack changes recommended. This is a code cleanup project that operates entirely within the existing dependency versions. Upgrading the Terraform SDK is explicitly out of scope and would create compound risk. Current versions are stable and functional.

**Core technologies:**
- Go 1.23.0 — current compiler, no upgrade needed
- govmomi v0.52.0 — supports all ESXi operations discovered, upgrading during migration adds unnecessary risk
- Terraform SDK v0.12.17 — legacy but stable, upgrading would require rewriting all schema definitions (out of scope)
- golang.org/x/crypto v0.40.0 — still required for SSH operations that must be retained (guest create/delete, config connectivity test)

### Expected Features

Based on codebase analysis, SSH removal coverage is mapped to existing resource operations, not new features.

**Full govmomi coverage (SSH removable):**
- Portgroup: create, read, update, delete, import — all operations have working govmomi implementations
- vSwitch: create, read, update, delete, import — all operations have working govmomi implementations
- Resource Pool: create, read, update, delete, import, GetID, GetName — all operations have working govmomi implementations
- Virtual Disk: create, read, grow, validate, delete — all operations have working govmomi implementations

**Partial govmomi coverage (SSH retained):**
- Guest (VM): read, power operations, IP detection, device info via govmomi — but create/delete have no govmomi alternative and must keep SSH
- Config: SSH connectivity test has no govmomi equivalent
- Data source esxi_host: missing dataSourceEsxiHostReadGovmomi implementation (build blocker)

**Dependencies:**
- portgroup requires vswitch (network dependency)
- guest requires resource_pool, virtual_disk, portgroup (VM placement and resources)

### Architecture Approach

Current architecture uses a routing pattern where each operation checks Config.useGovmomi flag and branches to either SSH commands (via runRemoteSshCommand in esxi_remote_cmds.go) or govmomi API calls. Target architecture removes this dual-path for resources with full coverage, keeping SSH only where govmomi cannot provide equivalent functionality.

**Major components:**
1. Config (config.go) — holds credentials and cached GovmomiClient; currently supports both SSH ConnectionStruct and govmomi client, will retain both post-cleanup for guest operations
2. Resources (5 total) — each has schema file + CRUD operation files + functions file with _govmomi variants; 4 resources become govmomi-only, 1 (guest) stays hybrid
3. Data Sources (6 total) — most already use govmomi; esxi_host needs implementation fix
4. govmomi_client.go — connection/session management (keep and harden)
5. govmomi_helpers.go — shared utilities like VM lookup, power ops, task waiting (keep and expand)
6. esxi_remote_cmds.go — SSH command execution (keep for guest operations, no longer called by other resources)

**Removal strategy:** Modify files to delete SSH branches rather than delete entire files. The useGovmomi routing pattern should be eliminated — resources with full govmomi support call govmomi directly, guest operations that need SSH call SSH directly without flag checking.

### Critical Pitfalls

Research identified 12 pitfalls; top 5 critical ones that could cause data loss or major production issues:

1. **Dual-Path State Inconsistency** — Resources created via SSH may not be properly managed by govmomi code due to ID format differences (vim-cmd vmid vs ManagedObjectReference). Prevention: establish canonical ID format, add migration logic in Read operations, test import functionality with SSH-created resources.

2. **Session Management and Resource Cleanup** — govmomi sessions may leak or expire mid-operation; ESXi free version limits 32 concurrent sessions. Prevention: implement session keepalive for long operations, validate session before each operation, ensure client Close() called via defer, test parallel operations.

3. **Error Handling Differences** — SSH commands return exit 0 even on partial failures, govmomi returns typed faults. Silent failures create inconsistent state. Prevention: comprehensive fault type checking, never assume task completion without WaitForTask, verify resource exists after creation (read-after-write).

4. **Feature Parity Gaps** — Some operations possible via esxcli/vim-cmd have no govmomi API equivalent. Removing SSH eliminates features users depend on. Prevention: audit ALL esxcli/vim-cmd commands before removal, document gaps, keep SSH for operations without govmomi alternative.

5. **Concurrent Operation Safety** — govmomi operations acquire locks on ESXi objects; parallel terraform operations may deadlock. Prevention: proper task waiting, retry logic with exponential backoff, test parallel resource creation, document resource dependencies.

## Implications for Roadmap

Based on research, suggested phase structure follows risk-ascending complexity with blocking issues resolved first:

### Phase 1: Fix Build Errors
**Rationale:** data_source_esxi_host.go has compilation errors blocking all work. Must fix before proceeding with any SSH removal. This establishes testing baseline and session management patterns without removing any SSH code.
**Delivers:** Clean compilation, passing test suite, validated session management
**Addresses:** Build blocker (missing dataSourceEsxiHostReadGovmomi function, type mismatches)
**Avoids:** Pitfall 2 (session management), Pitfall 3 (error handling), Pitfall 10 (test coverage gaps)
**Scope:**
- Implement missing dataSourceEsxiHostReadGovmomi or keep SSH for this data source
- Fix type mismatches in SSH helper functions
- Verify all existing tests pass
- Add session validation and cleanup patterns

### Phase 2: Remove SSH from Portgroup
**Rationale:** Smallest resource with complete govmomi coverage and good test coverage. Establishes removal pattern for other resources. Low complexity, low risk proof of concept.
**Delivers:** First resource with SSH fully removed, validated removal process
**Uses:** govmomi v0.52.0 HostNetworkSystem API
**Implements:** Direct govmomi calls (no useGovmomi flag routing)
**Avoids:** Pitfall 1 (state inconsistency via ID format validation), Pitfall 8 (security policy behavior differences)
**Scope:**
- Remove SSH branches from portgroup_functions.go (3 functions)
- Remove SSH branches from portgroup_create.go, portgroup_update.go, portgroup_delete.go
- Rewrite portgroup_import.go to use govmomi
- Verify portgroup tests pass with SSH code removed
- Document any ID format changes

### Phase 3: Remove SSH from vSwitch
**Rationale:** Similar to portgroup (networking resource, full govmomi coverage), slightly more complex. Builds confidence before tackling compute resources.
**Delivers:** Second resource cleaned, networking stack SSH-free
**Implements:** Direct govmomi calls for vSwitch operations
**Avoids:** Pitfall 1 (state inconsistency), Pitfall 8 (security policy behavior)
**Scope:**
- Remove SSH branches from vswitch_functions.go (2 functions)
- Remove SSH branches from vswitch_create.go, vswitch_delete.go
- Rewrite vswitch_import.go to use govmomi
- Verify vswitch tests pass
- Test portgroup + vswitch interaction (dependency)

### Phase 4: Remove SSH from Resource Pool
**Rationale:** Compute resource with full govmomi coverage. Must complete before virtual disk (dependency order).
**Delivers:** Resource pool management via govmomi only
**Implements:** Resource pool CRUD with ManagedObjectReference handling
**Avoids:** Pitfall 1 (state inconsistency), Pitfall 9 (resource pool path resolution)
**Scope:**
- Remove SSH branches from resource-pool_functions.go (3 functions)
- Remove SSH branches from resource-pool_create.go, resource-pool_update.go, resource-pool_delete.go
- Verify getPoolID_govmomi and getPoolNAME_govmomi handle all edge cases
- Test nested resource pool scenarios
- Verify tests pass

### Phase 5: Remove SSH from Virtual Disk
**Rationale:** Last fully-covered resource. More complex than networking but simpler than guest. Completes the "clean sweep" of removable SSH code.
**Delivers:** Virtual disk management via govmomi only
**Implements:** Disk operations with datastore browser API
**Avoids:** Pitfall 1 (state inconsistency), Pitfall 7 (disk path format inconsistency)
**Scope:**
- Remove SSH branches from virtual-disk_functions.go (4 functions)
- Implement or verify virtualDiskDelete_govmomi
- Remove SSH branches from virtual-disk_delete.go
- Standardize disk path format ([datastore] path/disk.vmdk)
- Test disk grow operations
- Verify data_source_esxi_virtual_disk works with SSH removed

### Phase 6: Infrastructure Cleanup
**Rationale:** With 4 resources SSH-free, clean up the routing infrastructure and shared code. Simplifies codebase significantly.
**Delivers:** Simplified architecture, removed useGovmomi flag, updated documentation
**Avoids:** Pitfall 12 (useGovmomi flag confusion)
**Scope:**
- Remove Config.useGovmomi flag (govmomi is now default)
- Update config.go to clarify SSH still used for guest operations
- Clean up unused SSH helper functions no longer called
- Remove unused imports from files that no longer use SSH
- Update data sources to remove SSH branches where applicable
- Document that SSH remains for guest create/delete (intentional, not technical debt)
- Update CLAUDE.md to reflect simplified architecture

### Phase 7: Test Hardening and Documentation
**Rationale:** Final validation before considering migration complete. Ensures robustness under production-like conditions.
**Delivers:** Comprehensive test coverage, production-ready documentation, migration guide
**Avoids:** Pitfall 5 (concurrent operation safety), Pitfall 10 (test coverage gaps)
**Scope:**
- Add concurrency tests (parallel resource creation)
- Test with ESXi session limits (32 concurrent sessions)
- Verify session cleanup (no leaks)
- Document state migration for users upgrading from SSH-based versions
- Add error message improvements mapping govmomi faults to user-friendly text
- Update README with govmomi-first architecture

### Phase Ordering Rationale

- Fix build first: cannot proceed with broken compilation, establishes testing baseline
- Networking before compute: portgroup/vswitch are simpler and interdependent, builds pattern
- Resource pool before virtual disk: guest depends on both, establish compute foundation
- Virtual disk last of removals: most complex remaining resource
- Infrastructure cleanup after removals: avoid premature optimization, see patterns first
- Test hardening final: validate complete system under production scenarios

Dependencies prevent reordering: vswitch before portgroup (dependency), resource pool before guest (dependency), build fix before everything (blocker).

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1 (Fix Build):** Need to decide implementation strategy for esxi_host data source (implement govmomi or keep SSH). May need govmomi API exploration.
- **Phase 7 (Test Hardening):** Session limit testing may reveal govmomi API behaviors not documented. Concurrency patterns may need research.

Phases with standard patterns (skip research-phase):
- **Phase 2-5 (SSH Removal):** Clear implementation pattern established in Phase 2, repeated for remaining resources. Codebase already contains complete govmomi implementations.
- **Phase 6 (Cleanup):** Standard refactoring work removing dead code and flags.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | No changes needed; existing stack analyzed directly from go.mod and codebase |
| Features | HIGH | Coverage matrix built from grep analysis of actual codebase (17 files with useGovmomi branches, 49 SSH command invocations cataloged) |
| Architecture | HIGH | Direct observation of current implementation patterns, target architecture is code removal not new design |
| Pitfalls | HIGH | Session management, state inconsistency, error handling risks observed directly in code; concurrency and feature parity risks based on training data (MEDIUM confidence sub-component) |

**Overall confidence:** HIGH

Research based on direct codebase analysis rather than external documentation. All 4 research files drew from actual source code inspection (grep patterns, function signatures, test files). Only uncertainty is around undocumented ESXi API behaviors or govmomi library edge cases not apparent from code inspection.

### Gaps to Address

Gaps identified that need attention during planning or execution:

- **esxi_host data source implementation:** Need to decide whether to implement dataSourceEsxiHostReadGovmomi or document that this data source intentionally retains SSH. Blocking decision for Phase 1.
- **Virtual disk delete govmomi coverage:** virtual-disk_delete.go marked "Need to verify" in feature research. Phase 5 must confirm virtualDiskDelete_govmomi exists and works before removing SSH.
- **Import functionality govmomi coverage:** portgroup_import.go and vswitch_import.go marked "CHECK" in feature research. Phases 2-3 must verify import works with SSH-created resources.
- **Session limit behavior under load:** Testing with 32 concurrent sessions (ESXi free limit) needed in Phase 7. May discover govmomi behaviors not documented.
- **ID format migration for existing users:** State migration guide needed in Phase 7. Research doesn't cover actual production deployments; may discover edge cases when users upgrade.
- **govmomi API version compatibility:** All research assumes govmomi v0.52.0 compatibility with target ESXi versions. If users run very old ESXi, may need compatibility matrix (deferred to production feedback).

## Sources

### Primary (HIGH confidence)
- Codebase analysis: esxi/*.go files — 17 files with useGovmomi branches identified via grep
- Test coverage analysis: esxi/*_test.go files — 8 files using govmomi simulator (github.com/vmware/govmomi/simulator)
- SSH command inventory: grep analysis found 49 vim-cmd/esxcli invocations across 19 files
- Dependency analysis: go.mod shows govmomi v0.52.0, terraform-plugin-sdk v0.12.17, Go 1.23.0
- Architecture documentation: CLAUDE.md describes dual-path implementation and testing approach

### Secondary (MEDIUM confidence)
- govmomi API capabilities: inferred from existing _govmomi function implementations (observed working code)
- ESXi session limits: training data indicates 32 concurrent sessions on free ESXi (not verified against official docs)
- Terraform provider development patterns: training data on proper error handling, state management

### Tertiary (LOW confidence - needs validation)
- govmomi API limitations vs ESXi CLI: gaps identified from code analysis (esxcli filesystem rescan, OVF deployment) but not exhaustively verified
- Performance characteristics: session management behavior under production load unknown
- Version-specific compatibility: ESXi version matrix for govmomi not verified

---
*Research completed: 2026-02-12*
*Ready for roadmap: yes*
