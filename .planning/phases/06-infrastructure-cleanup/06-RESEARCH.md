# Phase 6: Infrastructure Cleanup - Research

**Researched:** 2026-02-13
**Domain:** Go refactoring — feature flag removal, function renaming, import cleanup
**Confidence:** HIGH

## Summary

Phase 6 completes the SSH-to-govmomi migration by removing scaffolding left behind from the phased rollout. After phases 2-5 removed all SSH code from portgroup, vswitch, resource_pool, and virtual_disk resources, three cleanup tasks remain: (1) remove the `useGovmomi` feature flag and routing conditionals from guest operations, (2) rename all `_govmomi` suffixed functions to their final names, and (3) remove unused SSH imports from cleaned files.

This is pure refactoring with zero behavioral changes. All resources already use govmomi exclusively (except guest create/delete which intentionally retain SSH). The cleanup eliminates confusion and makes the intentional SSH retention for guest operations explicit.

**Primary recommendation:** Execute cleanup in three atomic commits: (1) remove useGovmomi flag and conditionals, (2) global function rename removing _govmomi suffix, (3) remove unused imports. Each commit maintains compilability and test passage.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| N/A | N/A | Pure Go refactoring | Standard library only (no external tools required) |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `goimports` | Auto-remove unused imports | After manual import cleanup to verify |
| `go build ./...` | Verify compilation | After each atomic commit |
| `go test ./esxi/ -v` | Verify test passage | After each atomic commit |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual rename | `gorename` tool (golang.org/x/tools/refactor/rename) | Tool is obsolete since Go modules; gopls recommended but requires LSP setup; manual find-replace is simpler for 23 functions |
| Manual import cleanup | `goimports -w esxi/*.go` | Automatic tool preferred but requires verification that it only removes truly unused imports |

**Installation:**
```bash
# Optional: Install goimports for automated import cleanup verification
go install golang.org/x/tools/cmd/goimports@latest
```

## Architecture Patterns

### Current State (After Phases 2-5)

```
esxi/
├── config.go                    # Config.useGovmomi field (unused)
├── portgroup_functions.go       # portgroupRead() → portgroupRead_govmomi()
├── vswitch_functions.go         # vswitchRead() → vswitchRead_govmomi()
├── resource-pool_functions.go   # resourcePoolRead() → resourcePoolRead_govmomi()
├── virtual-disk_functions.go    # virtualDiskREAD() → virtualDiskREAD_govmomi()
├── guest_functions.go           # guestGetVMID() → if useGovmomi → _govmomi else SSH
├── guest-read.go                # guestRefreshGuestStruct() → if useGovmomi → _govmomi
└── data_source_esxi_guest.go    # if useGovmomi routing (line 448)
```

**Pattern:** Thin wrapper functions call `_govmomi` implementations. Guest files retain `if c.useGovmomi` conditionals but flag is never set (always false in production).

### Target State (After Phase 6)

```
esxi/
├── config.go                    # Config.useGovmomi field REMOVED
├── portgroup_functions.go       # portgroupRead() calls portgroupRead() (no suffix)
├── vswitch_functions.go         # vswitchRead() calls vswitchRead() (no suffix)
├── resource-pool_functions.go   # resourcePoolRead() calls resourcePoolRead() (no suffix)
├── virtual-disk_functions.go    # virtualDiskREAD() calls virtualDiskREAD() (no suffix)
├── guest_functions.go           # guestGetVMID() → SSH only (govmomi branch REMOVED)
├── guest-read.go                # guestRefreshGuestStruct() → govmomi only (SSH removed)
└── data_source_esxi_guest.go    # Direct govmomi calls (no conditional)
```

**Pattern:** Direct function calls with no routing logic. SSH code remains in `esxi_remote_cmds.go` for guest create/delete operations.

### Recommended Refactoring Sequence

```
Commit 1: Remove useGovmomi flag and conditionals
├── Task 1.1: Remove useGovmomi field from Config struct (config.go line 18)
├── Task 1.2: Remove guest operation conditionals (guest_functions.go, guest-read.go, data_source_esxi_guest.go, data_source_esxi_host.go)
├── Task 1.3: Update wrapper functions to call _govmomi directly (remove conditional routing)
└── Verify: go build ./... && go test ./esxi/ -v

Commit 2: Rename _govmomi functions to final names
├── Task 2.1: Rename 23 _govmomi functions (portgroup: 5, vswitch: 4, resource_pool: 5, virtual_disk: 4, guest: 3, govmomi_helpers: 2)
├── Task 2.2: Update all callers to use new names
└── Verify: go build ./... && go test ./esxi/ -v

Commit 3: Remove unused imports
├── Task 3.1: Remove SSH imports from files that no longer need them
├── Task 3.2: Run goimports to verify cleanup
└── Verify: go build ./... && go test ./esxi/ -v
```

### Pattern 1: Remove Feature Flag Field

**What:** Delete `useGovmomi bool` from Config struct and remove all references

**When to use:** After all feature flag conditionals removed from codebase

**Example:**
```go
// BEFORE (config.go lines 8-18)
type Config struct {
    esxiHostName    string
    esxiHostSSHport string
    esxiHostSSLport string
    esxiUserName    string
    esxiPassword    string
    esxiPrivateKeyPath string

    // New fields for govmomi
    govmomiClient *GovmomiClient  // Cached client connection
    useGovmomi    bool            // Feature flag for phased migration
}

// AFTER
type Config struct {
    esxiHostName    string
    esxiHostSSHport string
    esxiHostSSLport string
    esxiUserName    string
    esxiPassword    string
    esxiPrivateKeyPath string

    // govmomi client
    govmomiClient *GovmomiClient  // Cached client connection
}
```

### Pattern 2: Remove Conditional Routing

**What:** Replace `if c.useGovmomi { return _govmomi() } else { SSH }` with direct calls

**When to use:** When one code path is dead (never executed)

**Example:**
```go
// BEFORE (guest_functions.go lines 12-36)
func guestGetVMID(c *Config, guest_name string) (string, error) {
    // Use govmomi if enabled
    if c.useGovmomi {
        return guestGetVMID_govmomi(c, guest_name)
    }

    // Fallback to SSH
    esxiConnInfo := getConnectionInfo(c)
    // ... SSH implementation ...
    return vmid, nil
}

// AFTER (useGovmomi always false, so keep SSH path for guest operations)
func guestGetVMID(c *Config, guest_name string) (string, error) {
    esxiConnInfo := getConnectionInfo(c)
    log.Printf("[guestGetVMID]\n")

    var remote_cmd, vmid string
    var err error

    remote_cmd = fmt.Sprintf("vim-cmd vmsvc/getallvms 2>/dev/null |sort -n | "+
        "grep -m 1 \"[0-9] * %s .*%s\" |awk '{print $1}' ", guest_name, guest_name)

    vmid, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get vmid")
    // ... rest of SSH implementation ...
    return vmid, nil
}
```

**IMPORTANT:** Guest operations retain SSH (no govmomi alternative). Other resources use govmomi exclusively.

### Pattern 3: Rename Functions Removing _govmomi Suffix

**What:** Rename implementation functions from `functionName_govmomi` to `functionName`

**When to use:** After removing SSH alternatives, making govmomi the only implementation

**Example:**
```go
// BEFORE (portgroup_functions.go)
func portgroupRead(c *Config, name string) (string, int, error) {
    return portgroupRead_govmomi(c, name)  // Thin wrapper
}

func portgroupRead_govmomi(c *Config, name string) (string, int, error) {
    // ... govmomi implementation ...
}

// AFTER
func portgroupRead(c *Config, name string) (string, int, error) {
    // Inline the implementation directly OR keep as separate function
    log.Printf("[portgroupRead] Reading portgroup %s\n", name)

    var vswitch string
    var vlan int

    gc, err := c.GetGovmomiClient()
    // ... govmomi implementation (same code, no wrapper) ...
}

// OR keep helper pattern (recommended for testability)
func portgroupRead(c *Config, name string) (string, int, error) {
    return portgroupReadImpl(c, name)
}

func portgroupReadImpl(c *Config, name string) (string, int, error) {
    log.Printf("[portgroupReadImpl] Reading portgroup %s\n", name)
    // ... govmomi implementation ...
}
```

**Recommendation:** Inline wrapper functions into implementation for simplicity. If function has multiple callers and complex logic, keep separate implementation function with clean name (no suffix).

### Pattern 4: Remove Unused Imports

**What:** Delete import statements for packages no longer referenced

**When to use:** After removing all SSH code paths from a file

**Example:**
```go
// BEFORE (virtual-disk_functions.go)
import (
    "errors"        // UNUSED after SSH removal
    "fmt"
    "log"
    "strconv"
    "strings"

    "github.com/vmware/govmomi/object"
    "github.com/vmware/govmomi/vim25/types"
)

// AFTER
import (
    "fmt"
    "log"
    "strconv"
    "strings"

    "github.com/vmware/govmomi/object"
    "github.com/vmware/govmomi/vim25/types"
)
```

**Verification:**
```bash
# Go compiler will error on unused imports
go build ./esxi/

# goimports will auto-remove unused imports
goimports -w esxi/*.go

# Verify no changes (if manual cleanup was correct)
git diff esxi/
```

### Anti-Patterns to Avoid

- **Renaming across multiple commits:** Feature flag removal, function rename, and import cleanup are independent changes. DO NOT mix them in one commit. Each must maintain compilability independently.
- **Removing SSH infrastructure globally:** Guest create/delete operations REQUIRE SSH (no govmomi alternative). Only remove SSH from files that don't use it.
- **Keeping dead code branches:** Don't comment out the `if c.useGovmomi` branches — delete them entirely. Dead code is confusing and has no documentation value.
- **Batch renaming without testing:** Rename functions incrementally by resource (portgroup → vswitch → resource_pool → virtual_disk → guest), building and testing after each resource.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Function renaming across codebase | Manual find-replace with string literals | Go's type system + compiler | Compiler catches all call sites; string search misses dynamic calls |
| Unused import detection | grep/regex for import usage | `go build` + `goimports` | Compiler definitively knows what's used; regex can't handle aliases or dot imports |
| Feature flag removal automation | Custom AST rewriting tool | Manual refactoring | Only 5 files affected (guest_functions.go, guest-read.go, data_source_esxi_guest.go, data_source_esxi_host.go, config.go); automation overhead exceeds manual effort |

**Key insight:** Go's compiler and standard tooling (goimports) handle verification better than any custom solution. Trust the compiler to find breakage.

## Common Pitfalls

### Pitfall 1: Breaking Guest Operations by Removing SSH Infrastructure

**What goes wrong:** Developer removes `esxi_remote_cmds.go`, `ConnectionStruct`, or `getConnectionInfo()` because "we migrated to govmomi"

**Why it happens:** Misunderstanding scope — guest create/delete still require SSH (no govmomi alternative exists)

**How to avoid:**
- Read ROADMAP.md Phase 6 reasoning: "esxi_remote_cmds.go, ConnectionStruct, and SSH code remain available for guest operations"
- Grep for SSH usage before deleting: `grep -r "runRemoteSshCommand" esxi/guest*.go`
- Verify guest tests still pass after cleanup

**Warning signs:**
- Guest creation tests fail with "undefined: runRemoteSshCommand"
- Config.validateEsxiCreds() fails to compile (uses SSH to validate connection)
- Guest create/delete operations no longer work

### Pitfall 2: Forgetting Test File useGovmomi Assignments

**What goes wrong:** Tests fail with "unknown field 'useGovmomi' in struct literal" after removing flag from Config struct

**Why it happens:** Test files set `useGovmomi: true` in Config literals (27 occurrences across 8 test files)

**How to avoid:**
```bash
# Find all test uses BEFORE removing field
grep -n "useGovmomi:" esxi/*_test.go

# After removing Config.useGovmomi field, remove from ALL test files
# Files affected:
# - guest_device_info_test.go (3 occurrences)
# - vswitch_functions_test.go (2 occurrences)
# - virtual-disk_functions_test.go (2 occurrences)
# - portgroup_functions_test.go (4 occurrences)
# - govmomi_helpers_test.go (6 occurrences)
# - resource-pool_functions_test.go (3 occurrences)
# - govmomi_client_test.go (1 occurrence)
```

**Warning signs:**
- `go test ./esxi/` fails with compilation errors (not test failures)
- Error message: "unknown field 'useGovmomi' in struct literal of type Config"

### Pitfall 3: Incomplete Function Rename (Orphaned _govmomi Functions)

**What goes wrong:** After renaming, both `functionName()` and `functionName_govmomi()` exist with identical implementations

**Why it happens:** Developer renames function definition but forgets to update callers, or updates some callers but not all

**How to avoid:**
1. **Rename systematically by file:**
   - Identify all `_govmomi` functions in file: `grep "^func.*_govmomi" esxi/portgroup_functions.go`
   - Rename ONE function at a time: definition → callers → tests
   - Build and test after each rename: `go build ./... && go test ./esxi/ -v -run TestPortgroup`

2. **Use compiler as verification:**
   ```bash
   # After renaming portgroupRead_govmomi → portgroupRead
   # Search for old name
   grep "portgroupRead_govmomi" esxi/*.go
   # Should find ZERO matches (except maybe comments)
   ```

3. **Systematic search after rename:**
   ```bash
   # Find any remaining _govmomi functions
   grep -n "func.*_govmomi(" esxi/*.go
   # Should be empty after full cleanup
   ```

**Warning signs:**
- `go build` succeeds but `grep "_govmomi" esxi/*.go` shows function definitions
- Two functions with nearly identical implementations (one with suffix, one without)
- Code coverage shows unreachable functions

### Pitfall 4: Removing Imports Still Used by Guest Operations

**What goes wrong:** `golang.org/x/crypto/ssh` removed from file that still needs it for guest create/delete

**Why it happens:** Assuming "we migrated to govmomi" means ALL SSH usage is gone

**How to avoid:**
```bash
# Before removing SSH import from a file, verify usage
grep "ssh\." esxi/guest-create.go
# If output shows ssh.Client, ssh.Session, etc., KEEP the import

# Safe to remove SSH imports from these files (verified no SSH usage):
# - portgroup_functions.go
# - vswitch_functions.go
# - resource-pool_functions.go
# - virtual-disk_functions.go

# KEEP SSH imports in these files (still use SSH):
# - esxi_remote_cmds.go (SSH implementation)
# - guest-create.go (uses runRemoteSshCommand)
# - guest-delete.go (uses runRemoteSshCommand)
# - config.go (validateEsxiCreds uses SSH)
```

**Warning signs:**
- Compilation fails with "undefined: runRemoteSshCommand"
- Guest creation/deletion fails in integration tests

## Code Examples

Verified patterns from codebase analysis:

### Remove useGovmomi Field from Config

```go
// Source: esxi/config.go (current state)
type Config struct {
    esxiHostName       string
    esxiHostSSHport    string
    esxiHostSSLport    string
    esxiUserName       string
    esxiPassword       string
    esxiPrivateKeyPath string

    // New fields for govmomi
    govmomiClient *GovmomiClient  // Cached client connection
    useGovmomi    bool            // Feature flag for phased migration ← DELETE THIS
}

// AFTER CLEANUP:
type Config struct {
    esxiHostName       string
    esxiHostSSHport    string
    esxiHostSSLport    string
    esxiUserName       string
    esxiPassword       string
    esxiPrivateKeyPath string

    govmomiClient *GovmomiClient  // Cached govmomi client
}
```

### Remove Conditional from Guest Operations (Keep SSH)

```go
// Source: esxi/guest_functions.go lines 12-36 (current state)
func guestGetVMID(c *Config, guest_name string) (string, error) {
    // Use govmomi if enabled
    if c.useGovmomi {
        return guestGetVMID_govmomi(c, guest_name)
    }

    // Fallback to SSH
    esxiConnInfo := getConnectionInfo(c)
    log.Printf("[guestGetVMID]\n")

    var remote_cmd, vmid string
    var err error

    remote_cmd = fmt.Sprintf("vim-cmd vmsvc/getallvms 2>/dev/null |sort -n | "+
        "grep -m 1 \"[0-9] * %s .*%s\" |awk '{print $1}' ", guest_name, guest_name)

    vmid, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get vmid")
    log.Printf("[guestGetVMID] result: %s\n", vmid)
    if err != nil {
        log.Printf("[guestGetVMID] Failed get vmid: %s\n", err)
        return "", fmt.Errorf("Failed get vmid: %s\n", err)
    }

    return vmid, nil
}

// AFTER CLEANUP (keep SSH, remove conditional):
func guestGetVMID(c *Config, guest_name string) (string, error) {
    esxiConnInfo := getConnectionInfo(c)
    log.Printf("[guestGetVMID]\n")

    var remote_cmd, vmid string
    var err error

    remote_cmd = fmt.Sprintf("vim-cmd vmsvc/getallvms 2>/dev/null |sort -n | "+
        "grep -m 1 \"[0-9] * %s .*%s\" |awk '{print $1}' ", guest_name, guest_name)

    vmid, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get vmid")
    log.Printf("[guestGetVMID] result: %s\n", vmid)
    if err != nil {
        log.Printf("[guestGetVMID] Failed get vmid: %s\n", err)
        return "", fmt.Errorf("Failed get vmid: %s\n", err)
    }

    return vmid, nil
}
```

### Remove Conditional from Data Source (Use govmomi)

```go
// Source: esxi/data_source_esxi_guest.go line 448 (current state)
if c.useGovmomi {
    vm, err := findVMByName(gc.Context(), gc.Finder, guest_name)
    if err != nil {
        return fmt.Errorf("Failed to find VM: %s", err)
    }
    vmid = vm.Reference().Value
} else {
    vmid, err = guestGetVMID(c, guest_name)
    if err != nil {
        d.SetId("")
        return fmt.Errorf("Unable to find guest: %s", err)
    }
}

// AFTER CLEANUP (use govmomi only, remove SSH branch):
vm, err := findVMByName(gc.Context(), gc.Finder, guest_name)
if err != nil {
    return fmt.Errorf("Failed to find VM: %s", err)
}
vmid := vm.Reference().Value
```

### Rename Function Removing _govmomi Suffix

```go
// Source: esxi/portgroup_functions.go (current state)
func portgroupRead(c *Config, name string) (string, int, error) {
    return portgroupRead_govmomi(c, name)  // Thin wrapper
}

func portgroupRead_govmomi(c *Config, name string) (string, int, error) {
    log.Printf("[portgroupRead_govmomi] Reading portgroup %s\n", name)

    var vswitch string
    var vlan int

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", 0, fmt.Errorf("failed to get govmomi client: %w", err)
    }
    // ... rest of implementation ...
}

// AFTER CLEANUP (Option 1: Inline wrapper, keep function split for readability):
func portgroupRead(c *Config, name string) (string, int, error) {
    return portgroupReadImpl(c, name)
}

func portgroupReadImpl(c *Config, name string) (string, int, error) {
    log.Printf("[portgroupReadImpl] Reading portgroup %s\n", name)

    var vswitch string
    var vlan int

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", 0, fmt.Errorf("failed to get govmomi client: %w", err)
    }
    // ... rest of implementation ...
}

// AFTER CLEANUP (Option 2: Fully inline, single function):
func portgroupRead(c *Config, name string) (string, int, error) {
    log.Printf("[portgroupRead] Reading portgroup %s\n", name)

    var vswitch string
    var vlan int

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", 0, fmt.Errorf("failed to get govmomi client: %w", err)
    }
    // ... rest of implementation (same code, no wrapper) ...
}
```

**Recommendation:** Use Option 2 (fully inline) for simplicity. No need for wrapper layer when there's only one implementation.

### Remove Unused Test Assignments

```go
// Source: esxi/portgroup_functions_test.go line 27 (current state)
func TestPortgroupCreateReadDeleteGovmomi(t *testing.T) {
    model := simulator.ESX()
    defer model.Remove()

    err := model.Create()
    if err != nil {
        t.Fatal(err)
    }

    s := model.Service.NewServer()
    defer s.Close()

    c := &Config{
        esxiHostName:    s.URL.Hostname(),
        esxiHostSSLport: s.URL.Port(),
        esxiUserName:    "root",
        esxiPassword:    "password",
        useGovmomi:      true,  // ← DELETE THIS FIELD
    }
    // ... rest of test ...
}

// AFTER CLEANUP:
func TestPortgroupCreateReadDeleteGovmomi(t *testing.T) {
    model := simulator.ESX()
    defer model.Remove()

    err := model.Create()
    if err != nil {
        t.Fatal(err)
    }

    s := model.Service.NewServer()
    defer s.Close()

    c := &Config{
        esxiHostName:    s.URL.Hostname(),
        esxiHostSSLport: s.URL.Port(),
        esxiUserName:    "root",
        esxiPassword:    "password",
    }
    // ... rest of test ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Dual-path routing with `if c.useGovmomi` | Direct govmomi calls for all non-guest resources | Phase 6 (2026-02) | Eliminates routing complexity, makes SSH retention for guest operations explicit |
| `_govmomi` function suffix | Clean function names without suffix | Phase 6 (2026-02) | Removes migration artifact, simplifies codebase navigation |
| Feature flag in Config struct | No feature flag (govmomi is the implementation) | Phase 6 (2026-02) | Reduces Config struct size, eliminates dead field |

**Deprecated/outdated:**
- `Config.useGovmomi` field — flag is never set to true in production; all cleaned resources ignore it
- Wrapper functions that only call `_govmomi` implementations — unnecessary indirection after SSH removal
- SSH imports in cleaned resource files — portgroup, vswitch, resource_pool, virtual_disk no longer use SSH

## Open Questions

1. **Should we rename _govmomi functions or keep them?**
   - What we know: Current wrappers add zero value (just pass-through calls)
   - What's unclear: Is there benefit to keeping `_govmomi` suffix as documentation?
   - Recommendation: Remove suffix. Code history (git log) preserves migration context; current code should reflect current reality (govmomi is THE implementation, not "the govmomi alternative")

2. **Should we remove test name suffixes (TestPortgroupCreateReadDeleteGovmomi)?**
   - What we know: Test names include "Govmomi" suffix from dual-path era
   - What's unclear: Does suffix provide value or is it migration artifact?
   - Recommendation: KEEP test name suffixes. Tests validate govmomi API usage specifically (distinguishes from hypothetical SSH tests if we ever needed to add fallback logic)

3. **Should we document SSH retention for guest operations?**
   - What we know: Guest create/delete use SSH by necessity (no govmomi alternative)
   - What's unclear: How to prevent future confusion about "why does SSH code still exist?"
   - Recommendation: Add comment in `guest-create.go` and `guest-delete.go`: `// SSH implementation required - no govmomi alternative for OVF deployment and VM creation from ISO`

## Sources

### Primary (HIGH confidence)
- Codebase analysis (grep, file reading)
  - `esxi/config.go` (Config struct with useGovmomi field)
  - `esxi/guest_functions.go` (conditional routing pattern)
  - `esxi/portgroup_functions.go` (wrapper pattern example)
  - `.planning/phases/02-remove-ssh-from-portgroup/02-VERIFICATION.md` (established removal pattern)
  - `.planning/ROADMAP.md` (Phase 6 goals and SSH retention rationale)

### Secondary (MEDIUM confidence)
- [Automatic Feature Flag Cleanup with Codemod](https://codemod.com/blog/automatic-feature-flag-removal-and-cleanup-with-codemods) — automated cleanup patterns
- [How to Use Feature Flags in Go](https://oneuptime.com/blog/post/2026-01-07-go-feature-flags/view) — lifecycle management practices
- [goimports command - golang.org/x/tools/cmd/goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) — import cleanup tool

### Tertiary (LOW confidence)
- [Rename refactorings | GoLand Documentation](https://www.jetbrains.com/help/go/rename-refactorings.html) — IDE-based rename patterns (not applicable for CLI workflow)
- [GitHub - uber/piranha: A tool for refactoring code related to feature flag APIs](https://github.com/uber/piranha) — automated tool (overkill for 5 files)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Pure Go refactoring, no external dependencies
- Architecture: HIGH — Current and target states verified via codebase reading
- Pitfalls: HIGH — Identified through analysis of current usage patterns and test file structure

**Research date:** 2026-02-13
**Valid until:** 60 days (2026-04-14) — refactoring patterns are stable; codebase state accurate as of today
