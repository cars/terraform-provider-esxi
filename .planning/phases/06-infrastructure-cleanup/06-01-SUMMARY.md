---
phase: 06-infrastructure-cleanup
plan: 01
subsystem: infrastructure
tags:
  - cleanup
  - feature-flag-removal
  - dead-code-elimination
dependency_graph:
  requires:
    - phase: 05-remove-ssh-from-virtual-disk
      plan: 01
      reason: All SSH removal complete before cleanup
  provides:
    - artifact: Config struct without useGovmomi field
      consumers: [all resources, all data sources, all tests]
    - artifact: Direct function calls (no conditionals)
      consumers: [guest operations, data sources]
  affects:
    - esxi/config.go
    - esxi/guest_functions.go
    - esxi/guest-read.go
    - esxi/data_source_esxi_guest.go
    - esxi/data_source_esxi_host.go
    - all test files
tech_stack:
  added: []
  patterns:
    - Direct function calls instead of conditional routing
    - SSH as primary implementation for guest operations
    - govmomi as primary implementation for data sources
key_files:
  created: []
  modified:
    - path: esxi/config.go
      description: Removed useGovmomi field and updated comments
      lines_changed: 3
    - path: esxi/guest_functions.go
      description: Removed 6 useGovmomi conditionals, kept SSH implementations
      lines_changed: 42
    - path: esxi/guest-read.go
      description: Removed useGovmomi conditional from guestREAD
      lines_changed: 7
    - path: esxi/data_source_esxi_guest.go
      description: Made device info reading unconditional
      lines_changed: 3
    - path: esxi/data_source_esxi_host.go
      description: Simplified to direct govmomi call
      lines_changed: 6
    - path: esxi/*_test.go (7 files)
      description: Removed useGovmomi from all test Config literals
      lines_changed: 27
decisions:
  - decision: Keep SSH implementation for guest operations
    rationale: No govmomi alternatives exist for guest create/delete/power management
    impact: SSH infrastructure remains for guest VM lifecycle operations
  - decision: Keep govmomi implementation for data sources
    rationale: Data sources already use govmomi exclusively with better performance
    impact: Data sources bypass SSH entirely
  - decision: Remove useGovmomi field entirely
    rationale: No more dual-path code after Phases 2-5 completion
    impact: Simpler Config struct, no feature flag checks
  - decision: Keep dataSourceEsxiHostReadSSH function (unused)
    rationale: Go allows unused non-exported functions; removal would require removing all its SSH helpers
    impact: No compilation errors, dead code can be cleaned in future
metrics:
  duration_seconds: 217
  tasks_completed: 2
  files_modified: 12
  commits: 2
  test_baseline: "27/32 passing (maintained)"
  completed_at: "2026-02-14T04:50:17Z"
---

# Phase 6 Plan 1: Remove useGovmomi Feature Flag

Eliminated the useGovmomi feature flag from Config struct and all conditional routing in guest operations and data sources, completing the SSH-to-govmomi migration infrastructure cleanup.

## Objective

Remove the useGovmomi feature flag that was preserved during Phases 2-5 SSH removal. After this plan, no code references useGovmomi anywhere. Guest operations use SSH directly (no govmomi alternative exists), and data sources use govmomi directly (preferred path).

## What Was Done

### Task 1: Remove useGovmomi Field from Config and All Test Files

**Changes:**
1. Removed `useGovmomi bool` field from Config struct (esxi/config.go line 18)
2. Updated comment from "New fields for govmomi" to "govmomi client"
3. Removed `useGovmomi: true,` from all 27 Config struct literals across 7 test files:
   - esxi/guest_device_info_test.go (3 occurrences)
   - esxi/vswitch_functions_test.go (2 occurrences)
   - esxi/portgroup_functions_test.go (4 occurrences)
   - esxi/govmomi_helpers_test.go (6 occurrences)
   - esxi/resource-pool_functions_test.go (3 occurrences)
   - esxi/govmomi_client_test.go (1 occurrence)
   - esxi/virtual-disk_functions_test.go (2 occurrences)

**Verification:**
- `grep -rn "useGovmomi" esxi/*_test.go esxi/config.go` returned zero matches
- Config struct compiles (expected errors in source files until Task 2)

**Commit:** 89d41a4

### Task 2: Clean useGovmomi Conditionals from Guest Operations and Data Sources

**Changes:**

1. **esxi/guest_functions.go** - Removed 6 conditionals, kept SSH path:
   - guestGetVMID (line 14): Deleted `if c.useGovmomi` block, kept SSH
   - guestValidateVMID (line 40): Deleted `if c.useGovmomi` block, kept SSH
   - guestPowerOn (line 407): Deleted `if c.useGovmomi` block, kept SSH
   - guestPowerOff (line 432): Deleted `if c.useGovmomi` block, kept SSH
   - guestPowerGetState (line 476): Deleted `if c.useGovmomi` block, kept SSH
   - guestGetIpAddress (line 503): Deleted `if c.useGovmomi` block, kept SSH
   - Removed all "Use govmomi if enabled" and "Fallback to SSH" comments

2. **esxi/guest-read.go** - Removed guestREAD conditional (line 91):
   - Deleted `if c.useGovmomi { return guestREAD_govmomi(...) }` block
   - Kept SSH implementation
   - Removed conditional comments

3. **esxi/data_source_esxi_guest.go** - Made device info unconditional (line 448):
   - Changed `if c.useGovmomi { ... }` to unconditional execution
   - guestReadDevices_govmomi now called unconditionally
   - All d.Set calls for device info executed unconditionally

4. **esxi/data_source_esxi_host.go** - Simplified routing (line 118):
   - Replaced function body with direct call: `return dataSourceEsxiHostReadGovmomi(d, c)`
   - Removed `if c.useGovmomi` conditional
   - Removed SSH fallback branch
   - Kept dataSourceEsxiHostReadSSH function (unused but safe in Go)

**Verification:**
- `grep -rn "useGovmomi" esxi/` returned zero matches (entire directory clean)
- `go build ./...` succeeded
- `go test ./esxi/ -v` showed 27/32 passing (baseline maintained)
- Guest functions still call runRemoteSshCommand (verified)
- Data sources still call govmomi APIs (verified)

**Commit:** add01cc

## Deviations from Plan

None - plan executed exactly as written.

## Test Results

**Baseline maintained:** 27/32 tests passing, 5 pre-existing simulator failures

**Passing tests (27):**
- All govmomi_client tests (7)
- All govmomi_helpers tests (6)
- All guest_device_info tests (3)
- Portgroup tests (3/4)
- Resource pool tests (2/3)
- Virtual disk tests (1/2)
- Vswitch tests (0/2 - known simulator limitations)
- Utility tests (5)

**Pre-existing failures (5):**
1. TestPortgroupUpdateGovmomi - simulator doesn't implement UpdateNetworkConfig
2. TestResourcePoolUpdateGovmomi - simulator doesn't implement UpdateResourcePool
3. TestVirtualDiskCreateReadGovmomi - simulator datastore browser limitation
4. TestVswitchCreateReadDeleteGovmomi - simulator returns zero for ports/mtu
5. TestVswitchUpdateGovmomi - simulator doesn't implement UpdateVirtualSwitch

All failures are simulator API coverage gaps, not code defects.

## Architecture Impact

**Before this plan:**
```go
// Config had feature flag
type Config struct {
    ...
    useGovmomi bool
}

// Functions had conditionals
func guestPowerOn(c *Config, vmid string) (string, error) {
    if c.useGovmomi {
        return guestPowerOn_govmomi(c, vmid)
    }
    // SSH fallback
}
```

**After this plan:**
```go
// Config clean
type Config struct {
    ...
    // no useGovmomi field
}

// Direct implementation
func guestPowerOn(c *Config, vmid string) (string, error) {
    esxiConnInfo := getConnectionInfo(c)
    // SSH implementation
}
```

**Benefits:**
- Simpler Config struct (1 fewer field)
- No conditional logic in CRUD functions (42 lines removed)
- Clearer code path (SSH for guests, govmomi for everything else)
- Zero feature flag checks across entire codebase
- Easier to understand for new contributors

**Data flow:**
- Guest operations → SSH via runRemoteSshCommand (guestGetVMID, guestPowerOn, etc.)
- Data sources → govmomi via GetGovmomiClient (dataSourceEsxiHostReadGovmomi, guestReadDevices_govmomi)
- Resources (non-guest) → govmomi wrappers from Phases 2-5

## Dead Code Note

The function `dataSourceEsxiHostReadSSH` remains in data_source_esxi_host.go but is never called. This is safe in Go (only unused imports cause errors). Removing it would require removing all its SSH helper functions (getHostInfoSSH, getDatastoreInfoSSH, etc.), which is beyond this plan's scope. It can be cleaned in a future dead-code elimination pass if desired.

## Success Criteria Met

- [x] useGovmomi field removed from Config struct
- [x] All 27 test file occurrences of useGovmomi removed
- [x] 6 conditionals in guest_functions.go cleaned (SSH retained)
- [x] 1 conditional in guest-read.go cleaned (SSH retained)
- [x] 1 conditional in data_source_esxi_guest.go cleaned (govmomi retained)
- [x] 1 conditional in data_source_esxi_host.go cleaned (govmomi retained)
- [x] Zero grep matches for "useGovmomi" in esxi/
- [x] go build ./... succeeds
- [x] go test ./esxi/ -v: 27/32 passing (baseline maintained)

## Next Steps

Phase 6 Plan 2 will rename `_govmomi` suffixed functions to their canonical names (e.g., `portgroupCreate_govmomi` → `portgroupCreate`) since they are now the only implementations.

---

## Self-Check: PASSED

**Commits verified:**
- FOUND: 89d41a4 (Task 1 - remove useGovmomi field from Config and tests)
- FOUND: add01cc (Task 2 - remove useGovmomi conditionals from source)

**Files verified:**
- FOUND: esxi/config.go
- FOUND: esxi/guest_functions.go
- FOUND: esxi/guest-read.go
- FOUND: esxi/data_source_esxi_guest.go
- FOUND: esxi/data_source_esxi_host.go
- FOUND: All 7 test files modified

**Verification checks:**
- PASS: Zero useGovmomi references in entire esxi/ directory
- PASS: go build ./... succeeds
- PASS: 27/32 tests passing (baseline maintained)

---

**Phase 6 Progress:** 1/4 plans complete
**Overall Progress:** 19/22 requirements complete (86%)
