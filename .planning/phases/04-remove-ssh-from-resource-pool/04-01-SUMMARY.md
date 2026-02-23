---
phase: 04-remove-ssh-from-resource-pool
plan: 01
subsystem: resource-pool
tags: [ssh-removal, refactor, govmomi]
dependency-graph:
  requires: [Phase 3 completion]
  provides: [SSH-free resource pool operations]
  affects: [data_source_esxi_resource_pool, guest VM creation with pool placement]
tech-stack:
  added: []
  patterns: [thin wrapper functions, direct govmomi routing]
key-files:
  created: []
  modified:
    - esxi/resource-pool_functions.go
    - esxi/resource-pool_create.go
    - esxi/resource-pool_update.go
    - esxi/resource-pool_delete.go
decisions:
  - "Keep wrapper function pattern established in Phase 2/3"
  - "Preserve resource-pool_import.go unchanged (already govmomi-based)"
  - "Defer _govmomi function renaming to Phase 6"
metrics:
  duration: 174
  completed: 2026-02-13T22:56:08Z
---

# Phase 4 Plan 1: Remove SSH from Resource Pool — Summary

**One-liner:** Resource pool operations now use govmomi API exclusively via thin wrapper pattern, eliminating all SSH dependencies.

## What Was Done

### Task 1: Remove SSH branches from resource pool CRUD functions
**Status:** Complete
**Commit:** 1e1dd3e

Removed all SSH code paths and SSH option-building logic from resource pool operations:

1. **resource-pool_functions.go** — Converted three functions to thin wrappers:
   - `getPoolID()` → calls `getPoolID_govmomi()` directly
   - `getPoolNAME()` → calls `getPoolNAME_govmomi()` directly
   - `resourcePoolRead()` → calls `resourcePoolRead_govmomi()` directly
   - Removed imports: `bufio`, `regexp` (SSH-only dependencies)
   - Kept imports: `fmt`, `log`, `strconv`, `strings` (used by govmomi implementations)

2. **resource-pool_create.go** — Direct govmomi routing:
   - Removed `esxiConnInfo` and `remote_cmd` variables
   - Removed all SSH option-building code (cpu_min_opt, cpu_min_expandable_opt, cpu_max_opt, cpu_shares_opt, mem_min_opt, mem_min_expandable_opt, mem_max_opt, mem_shares_opt)
   - Removed entire `if c.useGovmomi {...} else {...}` conditional
   - Now calls `resourcePoolCreate_govmomi()` directly
   - Removed import: `strconv` (only used for SSH option building)

3. **resource-pool_update.go** — Direct govmomi routing:
   - Removed `esxiConnInfo`, `remote_cmd`, and `stdout` variables
   - Removed SSH rename block (getPoolNAME check + vim-cmd rename command)
   - Removed all SSH option-building code (same 8 variables as create)
   - Removed entire `if c.useGovmomi {...} else {...}` conditional
   - Now calls `resourcePoolUpdate_govmomi()` directly (handles rename internally)
   - Kept import: `strings` (still used for resource_pool_name processing)

4. **resource-pool_delete.go** — Direct govmomi routing:
   - Removed entire `if c.useGovmomi {...} else {...}` conditional
   - Now calls `resourcePoolDelete_govmomi()` directly
   - No import changes (fmt and log still needed)

**Impact:**
- 4 files modified
- 348 lines removed (SSH code and option-building logic)
- 15 lines added (thin wrapper implementations)
- Build compiles cleanly

### Task 2: Verify import function and run full test suite
**Status:** Complete
**Commit:** None (verification only)

Verified resource pool implementation is SSH-free and functional:

1. **Import file verification:**
   - `resource-pool_import.go` already uses `getPoolNAME()` shared function
   - No SSH references found
   - Auto-fixed by Task 1 wrapper pattern (no changes needed)

2. **Data source verification:**
   - `data_source_esxi_resource_pool.go` calls `getPoolID()` and `resourcePoolRead()`
   - Auto-fixed by Task 1 wrapper pattern (no changes needed)

3. **Resource pool test results:**
   - ✅ TestResourcePoolCreateReadDeleteGovmomi: PASS
   - ❌ TestResourcePoolUpdateGovmomi: FAIL (known simulator limitation — vcsim does not implement Rename_Task)
   - ✅ TestGetPoolIDAndNameGovmomi: PASS

4. **Full test suite:**
   - **27/32 tests passing** (maintains Phase 3 baseline)
   - 5 failing tests (all pre-existing simulator limitations)
   - No regressions detected

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

### Build Check
```bash
go build ./...
```
✅ Compiles without errors

### SSH Elimination Check
```bash
grep -r "runRemoteSshCommand|getConnectionInfo|esxiConnInfo|vim-cmd|esxcli|remote_cmd" \
  esxi/resource-pool_*.go
```
✅ No SSH identifiers found in any resource pool file

### Wrapper Function Check
```bash
grep "return.*_govmomi" esxi/resource-pool_functions.go
```
✅ Three wrapper functions confirmed:
- `return getPoolID_govmomi(c, resource_pool_name)`
- `return getPoolNAME_govmomi(c, resource_pool_id)`
- `return resourcePoolRead_govmomi(c, pool_id)`

### Test Suite Check
```bash
go test ./esxi/ -v -run TestResourcePool
```
✅ Create/Read/Delete tests pass
❌ Update test fails (expected — Rename_Task not in simulator)

### Full Test Suite
```bash
go test ./esxi/ -v
```
✅ 27/32 passing (baseline maintained)

## What Works Now

1. **Resource pool operations use govmomi exclusively:**
   - Create: Uses `resourcePoolCreate_govmomi()` with complete allocation specs
   - Read: Uses `resourcePoolRead_govmomi()` for all pool properties
   - Update: Uses `resourcePoolUpdate_govmomi()` with rename support
   - Delete: Uses `resourcePoolDelete_govmomi()` for pool destruction
   - Lookup: Uses `getPoolID_govmomi()` and `getPoolNAME_govmomi()`

2. **Import functionality:**
   - `resource-pool_import.go` verifies pool exists via `getPoolNAME()` wrapper
   - Auto-routes to govmomi implementation

3. **Data source functionality:**
   - `data_source_esxi_resource_pool.go` uses `getPoolID()` and `resourcePoolRead()`
   - Auto-routes to govmomi implementations

4. **Guest VM integration:**
   - Guest create operations call `getPoolID()` for pool placement
   - Auto-routes to govmomi implementation
   - No changes needed to guest VM code

## Known Limitations

1. **Simulator update test failure:**
   - `TestResourcePoolUpdateGovmomi` fails because vcsim doesn't implement `Rename_Task`
   - This is a simulator limitation, not a code issue
   - Real ESXi hosts support rename operations via govmomi

2. **Other simulator limitations (pre-existing):**
   - VSwitch read doesn't populate ports/MTU
   - Portgroup/VSwitch update operations not implemented in simulator
   - Virtual disk create fails in simulator

## Architecture Impact

**Before:** Resource pool operations used dual-path (SSH vs govmomi) with complex branching logic

**After:** Resource pool operations route through thin wrappers to govmomi implementations only

**Pattern consistency:** Matches Phase 2 (portgroup) and Phase 3 (vswitch) wrapper approach

**Code reduction:** 348 lines of SSH code and option-building logic eliminated

## Next Steps

1. **Phase 5:** Remove SSH from virtual disk resource (4 requirements)
2. **Phase 6:** Infrastructure cleanup (remove useGovmomi flag, rename _govmomi functions, deprecate SSH helpers)

## Self-Check: PASSED

### Created files exist
No new files created in this phase.

### Modified files exist
```bash
[ -f "esxi/resource-pool_functions.go" ] && echo "FOUND: esxi/resource-pool_functions.go"
[ -f "esxi/resource-pool_create.go" ] && echo "FOUND: esxi/resource-pool_create.go"
[ -f "esxi/resource-pool_update.go" ] && echo "FOUND: esxi/resource-pool_update.go"
[ -f "esxi/resource-pool_delete.go" ] && echo "FOUND: esxi/resource-pool_delete.go"
```
✅ All 4 modified files exist

### Commits exist
```bash
git log --oneline --all | grep -q "1e1dd3e" && echo "FOUND: 1e1dd3e"
```
✅ Commit 1e1dd3e exists

---

**Phase 4 Plan 1 complete.** Resource pool operations are now SSH-free. Test baseline maintained (27/32 passing). Ready for Phase 5 (virtual disk SSH removal).
