---
phase: 02-remove-ssh-from-portgroup
plan: 01
subsystem: network
tags: [govmomi, portgroup, ssh-removal, api-migration]

# Dependency graph
requires:
  - phase: 01-fix-build-errors
    provides: Clean build and test baseline
provides:
  - SSH-free portgroup resource (create, read, update, delete, import)
  - Wrapper functions routing to govmomi implementations
  - Pattern for SSH removal in other resources
affects: [03-remove-ssh-from-vswitch, 04-remove-ssh-from-resource-pool, 05-remove-ssh-from-virtual-disk]

# Tech tracking
tech-stack:
  added: []
  patterns: [direct-govmomi-calls, wrapper-function-pattern]

key-files:
  created: []
  modified:
    - esxi/portgroup_functions.go
    - esxi/portgroup_create.go
    - esxi/portgroup_update.go
    - esxi/portgroup_delete.go
    - esxi/portgroup_import.go

key-decisions:
  - "Keep _govmomi function names during SSH removal (Phase 6 will rename)"
  - "Keep useGovmomi flag during SSH removal (Phase 6 will remove)"
  - "Create thin wrapper functions that directly call govmomi implementations"

patterns-established:
  - "SSH removal: delete conditional branches, create direct wrapper to _govmomi function"
  - "Import rewrite: use existing govmomi read functions instead of SSH commands"

# Metrics
duration: 131s
completed: 2026-02-13
---

# Phase 02 Plan 01: Remove SSH from Portgroup Summary

**Portgroup resource operates entirely via govmomi API with all SSH code paths removed and 3/4 tests passing**

## Performance

- **Duration:** 2 min 11 sec
- **Started:** 2026-02-13T20:25:53Z
- **Completed:** 2026-02-13T20:28:04Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Removed all SSH conditional branches from portgroup CRUD operations
- Simplified portgroupRead and portgroupSecurityPolicyRead to thin wrappers calling govmomi implementations
- Rewrote portgroup import to use govmomi-based portgroupRead function
- Cleaned up unused imports (regexp, strconv, strings, csvutil)
- Established SSH removal pattern for remaining resources

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove SSH branches from portgroup CRUD functions** - `5f59916` (refactor)
2. **Task 2: Rewrite portgroup import to use govmomi and verify all tests pass** - `6ee7712` (refactor)

## Files Created/Modified
- `esxi/portgroup_functions.go` - Removed SSH branches, now thin wrappers calling govmomi implementations
- `esxi/portgroup_create.go` - Direct call to portgroupCreate_govmomi, no SSH branch
- `esxi/portgroup_update.go` - Direct call to portgroupUpdate_govmomi, no SSH branch
- `esxi/portgroup_delete.go` - Direct call to portgroupDelete_govmomi, no SSH branch
- `esxi/portgroup_import.go` - Uses portgroupRead (govmomi) instead of SSH commands

## Decisions Made
- **Keep _govmomi suffix**: Function names remain with _govmomi suffix during SSH removal phase; Phase 6 (Infrastructure Cleanup) will handle renaming
- **Keep useGovmomi flag**: Config.useGovmomi flag remains in codebase during SSH removal phases; Phase 6 will remove it globally
- **Wrapper pattern**: Created thin wrapper functions (portgroupRead, portgroupSecurityPolicyRead) that directly call their _govmomi counterparts, allowing callers (resource read, data source) to work unchanged

## Deviations from Plan

None - plan executed exactly as written

## Issues Encountered

None - SSH removal proceeded cleanly with no unexpected complications

## Verification Results

**Build:** Clean compilation with no errors or warnings
**SSH Code:** Zero SSH identifiers found in any portgroup file (verified: runRemoteSshCommand, getConnectionInfo, esxiConnInfo, esxcli, remote_cmd)
**Govmomi Routing:** portgroupRead and portgroupSecurityPolicyRead correctly route to _govmomi implementations
**Tests:** 3/4 portgroup tests pass
  - PASS: TestPortgroupCreateReadDeleteGovmomi
  - PASS: TestPortgroupSecurityPolicyReadGovmomi
  - PASS: TestPortgroupNonExistentGovmomi
  - FAIL: TestPortgroupUpdateGovmomi (known simulator limitation - vcsim does not implement UpdatePortGroup)
**Regressions:** None - full test suite shows 27 passing, 5 failing (same as Phase 1 baseline)

## Pattern for Future Phases

This phase establishes the SSH removal pattern for Phases 3-5:

1. **Remove SSH branches from core functions:** Delete `if c.useGovmomi` conditionals and SSH else-branches
2. **Create thin wrappers:** Simplify functions to `return <function>_govmomi(c, ...)`
3. **Update imports:** Remove SSH-related imports (regexp, strconv, strings, csvutil) that become unused
4. **Rewrite import functions:** Replace SSH verification with calls to govmomi read functions
5. **Verify:** Build clean, no SSH identifiers remain, tests pass (accounting for simulator limitations)

## Next Phase Readiness

**Ready for Phase 3** (vSwitch SSH Removal):
- Portgroup SSH removal complete and verified
- Pattern established and documented
- Test baseline maintained (27/32 passing)
- No blockers or concerns

**Data source auto-fixed:** The data_source_esxi_portgroup.go file required no changes because it calls the shared portgroupRead and portgroupSecurityPolicyRead functions, which now route to govmomi automatically.

---
*Phase: 02-remove-ssh-from-portgroup*
*Completed: 2026-02-13*

## Self-Check: PASSED

All files and commits verified to exist.
