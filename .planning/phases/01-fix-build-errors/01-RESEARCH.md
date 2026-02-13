# Phase 1: Fix Build Errors - Research

**Researched:** 2026-02-12
**Domain:** Go compilation errors, type mismatches, govmomi API integration
**Confidence:** HIGH

## Summary

This phase fixes three categories of build errors in `esxi/data_source_esxi_host.go`:

1. **Missing function**: `dataSourceEsxiHostReadGovmomi` is referenced but not implemented
2. **Type mismatch**: Helper functions expect `map[string]string` but receive `ConnectionStruct`
3. **Type mismatch**: `runRemoteSshCommand` expects `ConnectionStruct` but receives `map[string]string`

The root cause is **inconsistent function signatures** in SSH helper functions. The fix requires implementing the missing govmomi function and correcting the function parameter types to match the existing `ConnectionStruct` pattern used throughout the codebase.

**Primary recommendation:** Implement `dataSourceEsxiHostReadGovmomi` using govmomi's property retrieval API and fix the SSH helper function signatures to accept `ConnectionStruct` instead of `map[string]string`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| govmomi | latest | VMware vSphere API client | Official VMware Go library, used across all resources |
| govmomi/simulator | latest | vCenter/ESXi simulator for testing | Bundled with govmomi, enables unit tests without real hardware |
| govmomi/vim25/mo | latest | Managed object types | Provides strongly-typed access to vSphere objects |
| govmomi/vim25/types | latest | vSphere API types | Defines all vSphere property structures |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| govmomi/property | latest | Property collector utilities | Efficient batch property retrieval |
| govmomi/object | latest | High-level object wrappers | Wraps raw API with convenience methods |
| context | stdlib | Operation cancellation/timeout | All govmomi API calls require context |

### Installation
Already present in `go.mod` - no new dependencies required.

## Architecture Patterns

### Recommended Project Structure
Already established in codebase:
```
esxi/
├── data_source_esxi_host.go      # Schema and routing logic
├── config.go                      # Config struct with govmomi client cache
├── govmomi_client.go              # Connection/session management
├── govmomi_helpers.go             # Shared govmomi utilities
├── esxi_remote_cmds.go            # SSH command execution
└── connection_info.go             # ConnectionStruct factory
```

### Pattern 1: Dual-Path Implementation (SSH + govmomi)

**What:** Resources/data sources check `c.useGovmomi` flag and route to appropriate implementation

**When to use:** All operations that can be implemented with either SSH or govmomi

**Example:**
```go
func dataSourceEsxiHostRead(d *schema.ResourceData, m interface{}) error {
    c := m.(*Config)

    if c.useGovmomi {
        return dataSourceEsxiHostReadGovmomi(d, c)
    }

    return dataSourceEsxiHostReadSSH(d, c)
}
```

**Established in:** `esxi/data_source_esxi_guest.go` lines 346-362

### Pattern 2: Property Retrieval with govmomi

**What:** Use `vm.Properties()` or `host.Properties()` to fetch managed object properties in a single API call

**When to use:** Reading host/VM configuration, hardware specs, runtime state

**Example:**
```go
// From govmomi examples and existing patterns
func getHostInfo(ctx context.Context, host *object.HostSystem) error {
    var mo mo.HostSystem
    err := host.Properties(ctx, host.Reference(), []string{
        "summary.hardware",
        "summary.config",
        "datastore",
    }, &mo)

    // Access properties:
    // mo.Summary.Hardware.MemorySize
    // mo.Summary.Config.Product.Version
    // mo.Summary.Hardware.CpuMhz
}
```

**Source:** govmomi examples/hosts/main.go (verified pattern)

### Pattern 3: ConnectionStruct for SSH Operations

**What:** Use `getConnectionInfo(c)` to create ConnectionStruct for SSH commands

**When to use:** All SSH fallback implementations

**Example:**
```go
func dataSourceEsxiHostReadSSH(d *schema.ResourceData, c *Config) error {
    esxiConnInfo := getConnectionInfo(c)

    // esxiConnInfo is ConnectionStruct, pass to SSH functions
    version, productName, uuid, err := getHostInfoSSH(esxiConnInfo)
}
```

**Established in:** `esxi/connection_info.go` lines 3-7

### Anti-Patterns to Avoid

- **Never use `map[string]string` for connection info**: The codebase uses `ConnectionStruct` consistently. Mixing types causes compilation errors.
- **Don't make multiple API calls for related properties**: Use property lists to fetch multiple properties in one call (e.g., `[]string{"summary.hardware", "summary.config"}`)
- **Don't skip context propagation**: All govmomi operations need context from `gc.Context()`

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Host property retrieval | Custom SSH parsing of `esxcli`/`dmidecode` | `mo.HostSystem` properties | govmomi provides strongly-typed access, handles version differences, less brittle |
| Datastore enumeration | Parse `esxcli storage filesystem list` | `mo.HostSystem.Datastore` property | Returns managed object references, includes inaccessible datastores, provides full metadata |
| Connection pooling | Custom SSH client cache | `Config.GetGovmomiClient()` | Already implemented with reconnect logic and session validation |
| Property batching | Multiple `host.Properties()` calls | Single call with property list | Reduces API roundtrips, improves performance |

**Key insight:** The govmomi API exposes the same underlying vSphere properties that SSH commands parse from text output. Using govmomi eliminates parsing fragility and version compatibility issues.

## Common Pitfalls

### Pitfall 1: Wrong Property Paths
**What goes wrong:** Using incorrect property names like `hardware.memorySize` instead of `summary.hardware.memorySize` causes nil pointer dereferences

**Why it happens:** vSphere property tree is deep and not well-documented

**How to avoid:**
- Use `govc object.collect -type HostSystem` to discover available properties
- Check `mo.HostSystem` struct definition in govmomi source
- Test with simulator first (simulator uses same property structure)

**Warning signs:** `panic: runtime error: invalid memory address or nil pointer dereference`

### Pitfall 2: Type Mismatch in Function Signatures
**What goes wrong:** Helper functions declared with `map[string]string` but called with `ConnectionStruct` (or vice versa)

**Why it happens:** Inconsistent refactoring when `ConnectionStruct` was introduced

**How to avoid:**
- Grep for function definitions before changing call sites
- Use ConnectionStruct consistently (already established pattern)
- Let compiler catch mismatches before running tests

**Warning signs:** `cannot use esxiConnInfo (variable of type ConnectionStruct) as map[string]string value`

### Pitfall 3: Missing Context Handling
**What goes wrong:** Operations hang indefinitely or don't respect cancellation

**Why it happens:** Forgetting to pass context or using `context.Background()` directly

**How to avoid:**
- Always use `gc.Context()` from GovmomiClient
- Add timeouts for long operations: `context.WithTimeout(ctx, 30*time.Minute)`
- Never create new background context when client context exists

**Warning signs:** Tests timeout, operations can't be cancelled

### Pitfall 4: Simulator vs Real ESXi Differences
**What goes wrong:** Tests pass with `simulator.VPX()` but fail on real ESXi

**Why it happens:** Simulator uses `VPX()` (vCenter) model, but provider targets standalone ESXi

**How to avoid:**
- Use `simulator.ESX()` for standalone ESXi simulation (CRITICAL for this provider)
- Verify properties exist before accessing (some properties are vCenter-only)
- Test datacenter name is "ha-datacenter" on standalone ESXi

**Warning signs:** Test passes but user reports "datacenter not found" or missing properties

## Code Examples

Verified patterns from official sources and existing codebase:

### Retrieving Host Hardware Information (govmomi)
```go
// Based on: govmomi examples/hosts/main.go + codebase patterns
func dataSourceEsxiHostReadGovmomi(d *schema.ResourceData, c *Config) error {
    gc, err := c.GetGovmomiClient()
    if err != nil {
        return fmt.Errorf("Failed to get govmomi client: %s", err)
    }

    ctx := gc.Context()

    // Get host system (standalone ESXi has one default host)
    host, err := getHostSystem(ctx, gc.Finder)
    if err != nil {
        return fmt.Errorf("Failed to get host system: %s", err)
    }

    // Retrieve properties in a single API call
    var mo mo.HostSystem
    err = host.Properties(ctx, host.Reference(), []string{
        "summary.hardware",
        "summary.config",
        "hardware.systemInfo",
        "datastore",
    }, &mo)
    if err != nil {
        return fmt.Errorf("Failed to get host properties: %s", err)
    }

    // Access strongly-typed properties
    // mo.Summary.Hardware.MemorySize (int64 bytes)
    // mo.Summary.Hardware.NumCpuCores (int16)
    // mo.Summary.Hardware.CpuMhz (int32)
    // mo.Summary.Hardware.Uuid (string)
    // mo.Summary.Config.Product.Version (string)
    // mo.Summary.Config.Product.FullName (string)
    // mo.Hardware.SystemInfo.Vendor (string - manufacturer)
    // mo.Hardware.SystemInfo.Model (string)

    // Set Terraform state
    d.SetId(c.esxiHostName)
    d.Set("hostname", c.esxiHostName)
    // ... set other fields from mo properties

    return nil
}
```

**Source:** govmomi/examples/hosts/main.go (lines showing property access pattern)

### Fixing SSH Helper Function Signatures
```go
// BEFORE (incorrect):
func getHostInfoSSH(esxiConnInfo map[string]string) (string, string, string, error) {
    stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get host version")
}

// AFTER (correct):
func getHostInfoSSH(esxiConnInfo ConnectionStruct) (string, string, string, error) {
    stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get host version")
}
```

**Rationale:** `runRemoteSshCommand` signature (line 71 of `esxi_remote_cmds.go`) is:
```go
func runRemoteSshCommand(esxiConnInfo ConnectionStruct, remoteSshCommand string, shortCmdDesc string) (string, error)
```

All callers must pass `ConnectionStruct`, not `map[string]string`.

### Using getConnectionInfo Pattern
```go
// Established pattern from esxi/connection_info.go
func dataSourceEsxiHostReadSSH(d *schema.ResourceData, c *Config) error {
    esxiConnInfo := getConnectionInfo(c)  // Returns ConnectionStruct

    // Pass ConnectionStruct to helper functions
    version, productName, uuid, err := getHostInfoSSH(esxiConnInfo)
    // ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SSH-only implementation | Dual-path (SSH + govmomi) | Ongoing migration | Better reliability, cleaner error handling |
| `map[string]string` for connection | `ConnectionStruct` | Introduced with govmomi support | Type safety, supports private key auth |
| Parse text output | Strongly-typed API | Ongoing migration | Eliminates parsing errors, version-independent |

**Deprecated/outdated:**
- Using `map[string]string` for connection parameters: Replaced by `ConnectionStruct` in `esxi_main.go` line 3-10
- Direct SSH commands where govmomi available: Provider is migrating to govmomi-first approach

## Open Questions

1. **Datastore capacity/free space via govmomi**
   - What we know: `mo.HostSystem.Datastore` provides datastore references
   - What's unclear: Does simulator populate datastore capacity or do we need additional property fetch?
   - Recommendation: Implement basic datastore list first, add capacity in separate property fetch if needed

2. **Simulator compatibility with ESX() vs VPX()**
   - What we know: Current tests use `simulator.VPX()` (vCenter model)
   - What's unclear: Will properties differ on `simulator.ESX()` (standalone ESXi model)?
   - Recommendation: Test with both, document any differences, prefer `ESX()` for this provider

3. **Hardware.SystemInfo vs Summary.Hardware**
   - What we know: Both contain hardware info, `Summary` is subset
   - What's unclear: Does `Summary.Hardware` include manufacturer/model/serial?
   - Recommendation: Fetch both property paths, use `Hardware.SystemInfo` for detailed hardware info

## Sources

### Primary (HIGH confidence)
- govmomi/esxi_remote_cmds.go (lines 71-98) - runRemoteSshCommand signature
- govmomi/connection_info.go (lines 3-7) - getConnectionInfo pattern
- govmomi/govmomi_helpers.go (lines 118-124) - getHostSystem helper
- govmomi/govmomi_client.go (lines 8-24, 88-94) - GovmomiClient structure
- govmomi/data_source_esxi_guest.go (lines 346-362) - Dual-path pattern
- govmomi/govmomi_client_test.go - Test patterns with simulator

### Secondary (MEDIUM confidence)
- [govmomi examples/hosts/main.go](https://github.com/vmware/govmomi/blob/main/examples/hosts/main.go) - Property retrieval pattern
- [govmomi mo.HostSystem struct](https://pkg.go.dev/github.com/vmware/govmomi/vim25/mo) - Available properties
- [govmomi HostSystem object](https://github.com/vmware/govmomi/blob/main/object/host_system.go) - Available methods

### Tertiary (LOW confidence)
- [govmomi govc USAGE.md](https://github.com/vmware/govmomi/blob/main/govc/USAGE.md) - Command examples showing property paths
- [govmomi Issue #798](https://github.com/vmware/govmomi/issues/798) - HostSystem hardware properties discussion
- [CormacHogan govc object.collect guide](https://cormachogan.com/2021/06/28/govc-object-collect-an-essential-tool-for-govmomi-developers/) - Property discovery techniques

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - govmomi already in use across codebase
- Architecture: HIGH - Patterns established in existing resources
- Pitfalls: HIGH - Direct observation of build errors and codebase conventions

**Research date:** 2026-02-12
**Valid until:** 2026-03-14 (30 days - stable library, established patterns)
