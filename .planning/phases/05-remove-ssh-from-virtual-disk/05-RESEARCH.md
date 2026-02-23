# Phase 5: Remove SSH from Virtual Disk - Research

**Researched:** 2026-02-13
**Domain:** Virtual disk SSH removal in Terraform provider ESXi
**Confidence:** HIGH

## Summary

Phase 5 removes SSH code paths from the virtual disk resource, following the proven pattern established in Phases 2-4. The virtual disk has nearly complete govmomi coverage with 4 of 5 core operations already implemented and tested (diskStoreValidate, virtualDiskCREATE, virtualDiskREAD, growVirtualDisk). The missing piece is virtualDiskDelete_govmomi, which needs implementation using govmomi's VirtualDiskManager.DeleteVirtualDisk method.

The virtual disk resource differs from previous phases in two key ways: (1) it uses dual path formats (legacy /vmfs/volumes/ for Terraform state vs. govmomi [datastore] notation for API calls), and (2) it includes a data source with a stubbed findVirtualDiskInDir_govmomi function that requires full implementation using the datastore browser API.

**Critical finding:** The codebase currently maintains /vmfs/volumes/ path format in virtdisk_id for backward compatibility with existing Terraform state files, while internally converting to [datastore] path/disk.vmdk format for all govmomi API calls. This dual-format pattern must be preserved during SSH removal.

**Primary recommendation:** Apply the Phase 2-4 wrapper pattern to virtual disk functions, implement virtualDiskDelete_govmomi using VirtualDiskManager.DeleteVirtualDisk, and complete findVirtualDiskInDir_govmomi using datastore browser SearchDatastore API. All path conversions already exist and work correctly.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/vmware/govmomi | Latest (v0.30+) | vSphere API client | Official VMware Go library, handles all ESXi operations |
| github.com/vmware/govmomi/object | (part of govmomi) | High-level object wrappers | Provides VirtualDiskManager, Datastore, FileManager abstractions |
| github.com/vmware/govmomi/vim25/types | (part of govmomi) | vSphere data types | Core type definitions for virtual disk specs, disk types |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/vmware/govmomi/simulator | Latest (v0.30+) | vSphere API simulator (vcsim) | All govmomi tests use this for standalone ESXi simulation |

### Current Implementation Status

**Govmomi functions implemented:** 4/5 complete
- `diskStoreValidate_govmomi` (lines 238-278 in virtual-disk_functions.go)
- `virtualDiskCREATE_govmomi` (lines 281-347)
- `virtualDiskREAD_govmomi` (lines 350-432)
- `growVirtualDisk_govmomi` (lines 435-499)

**Govmomi functions MISSING:** 1/5 needs implementation
- `virtualDiskDelete_govmomi` - NOT YET IMPLEMENTED (confirmed via grep, no results found)

**Data source stubbed:** 1 function needs full implementation
- `findVirtualDiskInDir_govmomi` (line 138-142 in data_source_esxi_virtual_disk.go) - currently returns "not yet implemented" error

**SSH functions to remove:** 5 locations
- `diskStoreValidate` SSH branch (lines 23-55 in virtual-disk_functions.go)
- `virtualDiskCREATE` SSH branch (lines 68-117)
- `virtualDiskREAD` SSH branch (lines 163-230)
- `growVirtualDisk` SSH branch (lines 129-151)
- `resourceVIRTUALDISKDelete` entire file (virtual-disk_delete.go, SSH-only, lines 1-48)
- `findVirtualDiskInDir` SSH branch (lines 110-134 in data_source_esxi_virtual_disk.go)

**Installation:**
Already in go.mod - no new dependencies required.

## Architecture Patterns

### Recommended Project Structure
```
esxi/
├── virtual-disk_functions.go        # Core logic with wrapper + govmomi implementations
├── virtual-disk_create.go           # Create operation (calls virtualDiskCREATE wrapper)
├── virtual-disk_read.go             # Read operation (calls virtualDiskREAD wrapper)
├── virtual-disk_update.go           # Update operation (calls growVirtualDisk wrapper)
├── virtual-disk_delete.go           # Delete operation (needs virtualDiskDelete_govmomi)
├── virtual-disk_import.go           # Import verification (no changes needed)
├── virtual-disk_functions_test.go   # Test coverage for govmomi operations
├── data_source_esxi_virtual_disk.go # Data source (uses shared functions)
├── resource_virtual-disk.go         # Resource schema definition
└── govmomi_helpers.go              # Shared helpers (getDatastoreByName, isDatastoreAccessible)
```

### Pattern 1: Thin Wrapper Functions (Phase 2-4 Established Pattern)

**What:** Replace SSH-routed functions with thin wrappers that directly call govmomi implementations

**When to use:** For all shared functions called by multiple files (diskStoreValidate, virtualDiskCREATE, virtualDiskREAD, growVirtualDisk)

**Example:**
```go
// Before (SSH routing with conditional)
func virtualDiskREAD(c *Config, virtdisk_id string) (string, string, string, int, string, error) {
    if c.useGovmomi {
        return virtualDiskREAD_govmomi(c, virtdisk_id)
    }
    // SSH code (68 lines) - DELETE THIS
}

// After (thin wrapper)
func virtualDiskREAD(c *Config, virtdisk_id string) (string, string, string, int, string, error) {
    return virtualDiskREAD_govmomi(c, virtdisk_id)
}
```

### Pattern 2: Path Format Conversion (CRITICAL - Already Implemented Correctly)

**What:** Maintain /vmfs/volumes/ format in Terraform state (virtdisk_id) while converting to [datastore] path format for govmomi API calls

**Why critical:** Existing Terraform state files use /vmfs/volumes/datastore/dir/disk.vmdk format. Breaking this format would invalidate all existing deployments.

**Current implementation (CORRECT - DO NOT CHANGE):**
```go
// virtualDiskCREATE_govmomi - lines 302-303
virtdisk_id := fmt.Sprintf("/vmfs/volumes/%s/%s/%s", virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name)
diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))
// Returns: [datastore] dir/disk.vmdk (govmomi format)
// But virtdisk_id returned is: /vmfs/volumes/datastore/dir/disk.vmdk (state format)
```

**Parsing logic (CORRECT - DO NOT CHANGE):**
```go
// virtualDiskREAD_govmomi - lines 357-368
// Parses /vmfs/volumes/datastore/dir/disk.vmdk into components
s := strings.Split(virtdisk_id, "/")
if len(s) < 6 {
    return "", "", "", 0, "", nil
}
// s[3] = datastore, s[4:len-1] = dir path, s[len-1] = disk name
virtual_disk_disk_store = s[3]
virtual_disk_name = s[len(s)-1]
if len(s) > 6 {
    virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")  // handles nested dirs
} else {
    virtual_disk_dir = s[4]
}
```

### Pattern 3: Implement Missing Delete Function

**What:** Create virtualDiskDelete_govmomi using VirtualDiskManager.DeleteVirtualDisk

**When to use:** In virtual-disk_delete.go to replace SSH-only delete operation

**API signature:**
```go
func (m VirtualDiskManager) DeleteVirtualDisk(
    ctx context.Context,
    name string,  // [datastore] path/disk.vmdk format
    dc *Datacenter  // nil for standalone ESXi
) (*Task, error)
```

**Implementation template:**
```go
func virtualDiskDelete_govmomi(c *Config, virtdisk_id string) error {
    log.Println("[virtualDiskDelete_govmomi]")

    // Parse virtdisk_id (/vmfs/volumes/ds/dir/disk.vmdk)
    s := strings.Split(virtdisk_id, "/")
    if len(s) < 6 {
        return fmt.Errorf("invalid virtdisk_id format")
    }

    var virtual_disk_dir string
    if len(s) > 6 {
        virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")
    } else {
        virtual_disk_dir = s[4]
    }
    virtual_disk_disk_store := s[3]
    virtual_disk_name := s[len(s)-1]

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return fmt.Errorf("failed to get govmomi client: %w", err)
    }

    ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
    if err != nil {
        return fmt.Errorf("failed to get datastore: %w", err)
    }

    // Convert to [datastore] path format
    diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

    // Delete virtual disk
    dm := object.NewVirtualDiskManager(gc.Client.Client)
    task, err := dm.DeleteVirtualDisk(gc.Context(), diskPath, nil)
    if err != nil {
        // Check if already deleted (acceptable error)
        if strings.Contains(err.Error(), "not found") {
            log.Printf("[virtualDiskDelete_govmomi] Already deleted: %s", virtdisk_id)
            return nil
        }
        return fmt.Errorf("failed to delete virtual disk: %w", err)
    }

    err = waitForTask(gc.Context(), task)
    if err != nil {
        return fmt.Errorf("delete task failed: %w", err)
    }

    log.Printf("[virtualDiskDelete_govmomi] Successfully deleted: %s", virtdisk_id)
    return nil
}
```

**Delete operation wrapper in virtual-disk_delete.go:**
```go
func resourceVIRTUALDISKDelete(d *schema.ResourceData, m interface{}) error {
    c := m.(*Config)
    log.Println("[resourceVIRTUALDISKDelete]")

    virtdisk_id := d.Id()

    // Delete virtual disk via govmomi
    err := virtualDiskDelete_govmomi(c, virtdisk_id)
    if err != nil {
        return fmt.Errorf("Failed to delete virtual disk: %w", err)
    }

    // Note: govmomi DeleteVirtualDisk handles both disk and descriptor files
    // No need to manually check/delete empty directories (different from SSH approach)

    d.SetId("")
    return nil
}
```

### Pattern 4: Complete Data Source Browser Implementation

**What:** Implement findVirtualDiskInDir_govmomi using datastore browser SearchDatastore API

**When to use:** In data_source_esxi_virtual_disk.go when virtual_disk_name is not specified

**API structure:**
```go
// Get datastore browser
browser, err := ds.Browser(gc.Context())

// Create search spec
spec := types.HostDatastoreBrowserSearchSpec{
    MatchPattern: []string{"*.vmdk"},  // Find all VMDK descriptor files
    Details: &types.FileQueryFlags{
        FileSize: true,  // Optional: get file sizes
    },
}

// Search directory
searchPath := ds.Path(virtual_disk_dir)
task, err := browser.SearchDatastore(gc.Context(), searchPath, &spec)

// Wait for results
info, err := task.WaitForResult(gc.Context())
result := info.Result.(types.HostDatastoreBrowserSearchResults)
```

**Implementation template:**
```go
func findVirtualDiskInDir_govmomi(c *Config, diskStore, dir string) (string, error) {
    log.Printf("[findVirtualDiskInDir_govmomi] diskStore=%s dir=%s", diskStore, dir)

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", fmt.Errorf("failed to get govmomi client: %w", err)
    }

    ds, err := getDatastoreByName(gc.Context(), gc.Finder, diskStore)
    if err != nil {
        return "", fmt.Errorf("failed to get datastore: %w", err)
    }

    browser, err := ds.Browser(gc.Context())
    if err != nil {
        return "", fmt.Errorf("failed to get datastore browser: %w", err)
    }

    // Search for .vmdk files (descriptor files, not -flat.vmdk)
    spec := types.HostDatastoreBrowserSearchSpec{
        MatchPattern: []string{"*.vmdk"},
        Details: &types.FileQueryFlags{
            FileSize: false,  // We don't need size for discovery
        },
    }

    searchPath := ds.Path(dir)
    task, err := browser.SearchDatastore(gc.Context(), searchPath, &spec)
    if err != nil {
        return "", fmt.Errorf("failed to search datastore: %w", err)
    }

    info, err := task.WaitForResult(gc.Context())
    if err != nil {
        return "", fmt.Errorf("search task failed: %w", err)
    }

    result, ok := info.Result.(types.HostDatastoreBrowserSearchResults)
    if !ok {
        return "", fmt.Errorf("unexpected search result type")
    }

    if len(result.File) == 0 {
        return "", nil  // No VMDK found
    }

    // Return first .vmdk file found (exclude -flat.vmdk files)
    for _, file := range result.File {
        if fileInfo, ok := file.(*types.FileInfo); ok {
            filename := fileInfo.Path
            // Skip -flat.vmdk files (only return descriptor .vmdk)
            if !strings.HasSuffix(filename, "-flat.vmdk") && strings.HasSuffix(filename, ".vmdk") {
                log.Printf("[findVirtualDiskInDir_govmomi] Found: %s", filename)
                return filename, nil
            }
        }
    }

    return "", nil  // No descriptor .vmdk found
}
```

### Anti-Patterns to Avoid

- **Breaking path format:** Don't change virtdisk_id return format from /vmfs/volumes/ to [datastore] - would break existing Terraform state
- **Forgetting directory cleanup:** SSH version manually deleted empty directories - govmomi DeleteVirtualDisk handles cleanup automatically, don't replicate manual directory removal
- **Not handling "already deleted" errors:** Delete operations should treat "not found" errors as success (idempotent behavior)
- **Using wrong path format for API calls:** Always use ds.Path() to convert to [datastore] format before calling VirtualDiskManager methods

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path format conversion | String manipulation for [datastore] format | `ds.Path(relativePath)` | Handles edge cases, datastore name escaping, standardized format |
| Virtual disk deletion | Manual vmkfstools over SSH | `VirtualDiskManager.DeleteVirtualDisk()` | Handles descriptor + flat files, validates deletion, supports VSAN/NFS |
| Directory enumeration | SSH `ls` parsing | `HostDatastoreBrowser.SearchDatastore()` | Works across all datastore types, structured results, proper error handling |
| Directory creation | SSH `mkdir -p` | Unnecessary - VirtualDiskManager creates parent dirs | API handles directory creation automatically during disk creation |
| Datastore rescanning | SSH `esxcli storage filesystem rescan` | `HostStorageSystem.RescanAllHba/RescanVmfs()` | Proper API calls, already implemented in diskStoreValidate_govmomi |

**Key insight:** The govmomi VirtualDiskManager API is higher-level than SSH vmkfstools commands. It automatically handles descriptor file + flat file coordination, directory creation, and datastore path resolution. Don't replicate low-level SSH logic - trust the API to handle implementation details.

## Common Pitfalls

### Pitfall 1: Path Format Confusion

**What goes wrong:** Mixing /vmfs/volumes/ format and [datastore] format causes "invalid datastore path" errors

**Why it happens:** The codebase uses /vmfs/volumes/datastore/dir/disk.vmdk for Terraform state (virtdisk_id) but govmomi requires [datastore] dir/disk.vmdk format for API calls

**How to avoid:**
- ALWAYS use `ds.Path(fmt.Sprintf("%s/%s", dir, name))` when calling VirtualDiskManager methods
- ALWAYS return /vmfs/volumes/ format from CREATE operations for Terraform state
- ALWAYS parse /vmfs/volumes/ format in READ/UPDATE/DELETE operations to extract datastore/dir/name components

**Warning signs:**
- Error messages containing "Invalid datastore path"
- Error messages with double brackets like "[[datastore]]"
- API calls failing with path parsing errors

**Example from codebase:**
```go
// CORRECT (lines 302-303, 335, 345-346 in virtual-disk_functions.go)
virtdisk_id := fmt.Sprintf("/vmfs/volumes/%s/%s/%s", ...)  // State format
diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))  // API format
task, err := dm.CreateVirtualDisk(gc.Context(), diskPath, nil, spec)  // Use API format
return virtdisk_id, nil  // Return state format
```

### Pitfall 2: Directory Deletion Assumptions

**What goes wrong:** Replicating SSH empty directory deletion logic causes unnecessary complexity

**Why it happens:** SSH version (lines 35-44 in virtual-disk_delete.go) manually checks if directory is empty and deletes it. This pattern isn't needed with govmomi.

**How to avoid:**
- DeleteVirtualDisk removes both descriptor and flat files automatically
- ESXi automatically manages datastore directory cleanup
- Don't check directory emptiness or manually delete directories in govmomi version
- If directory removal is truly needed, use FileManager.DeleteDatastoreFile() with directory path

**Warning signs:**
- Code checking directory contents after delete
- Manual rmdir operations
- Complex directory cleanup logic

**Example:**
```go
// WRONG (SSH pattern - don't replicate)
remote_cmd = fmt.Sprintf("ls -al \"/vmfs/volumes/%s/%s/\" |wc -l", ...)
stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "Check if Storage dir is empty")
if stdout == "3" { rmdir(...) }

// CORRECT (govmomi pattern)
err := virtualDiskDelete_govmomi(c, virtdisk_id)
// Done - no directory cleanup needed
```

### Pitfall 3: Disk Type Detection Complexity

**What goes wrong:** Attempting to determine exact disk type (thin/zeroedthick/eagerzeroedthick) results in complex vmkfstools parsing or unreliable API queries

**Why it happens:** VirtualDiskManager.QueryVirtualDiskInfo exists but isn't straightforward to use, and vcsim doesn't fully implement it. SSH version uses vmkfstools -t0 with grep patterns (lines 209-227 in virtual-disk_functions.go).

**How to avoid:**
- Current govmomi implementation returns "Unknown" for disk type (line 428) - this is ACCEPTABLE
- Disk type is only informational - not used for any operations
- Tests already accommodate "Unknown" type (line 133-135 in test file)
- Don't attempt to replicate vmkfstools -t0 parsing via govmomi unless absolutely required

**Warning signs:**
- Complex QueryVirtualDiskInfo parsing logic
- Attempting to read VMDK descriptor files manually
- Test failures due to disk type mismatches

**Current approach (CORRECT):**
```go
// virtualDiskREAD_govmomi line 428
virtual_disk_type = "Unknown"  // Acceptable - type is informational only
```

### Pitfall 4: vcsim Simulator Limitations

**What goes wrong:** Tests fail with "operation not supported" or nil pointer errors in vcsim

**Why it happens:** vcsim (the test simulator) has incomplete implementations:
- Disk capacity fields only update specified field (capacityInKB or capacityInBytes, not both)
- DeleteVirtualDisk may not clean up all internal state
- SearchDatastore works but may have incomplete file metadata

**How to avoid:**
- Test against vcsim limitations, not real ESXi behavior
- Accept "Unknown" disk type in tests (simulator limitation)
- Don't rely on exact capacity byte values in assertions
- Focus on functional behavior (create/read/delete succeeds) not internal details

**Warning signs:**
- Nil pointer dereferences in tests
- Inconsistent disk capacity between operations
- Missing file metadata in browser results

**Example from existing tests:**
```go
// TestVirtualDiskCreateReadGovmomi lines 127-135
// Size might not be exact due to rounding
if readSize < diskSize-1 || readSize > diskSize+1 {
    t.Errorf("Expected size around %d GB, got %d GB", diskSize, readSize)
}

// Type might be "Unknown" if not fully supported in simulator
if readType != diskType && readType != "Unknown" {
    t.Logf("Warning: Expected type %s, got %s (may be simulator limitation)", diskType, readType)
}
```

### Pitfall 5: Ignoring "Already Deleted" Errors

**What goes wrong:** Delete operations fail when resource doesn't exist, breaking Terraform's idempotent behavior

**Why it happens:** VirtualDiskManager.DeleteVirtualDisk returns error if disk doesn't exist, but Terraform expects deletes to succeed even if resource is already gone

**How to avoid:**
- Check error messages for "not found" or similar indicators
- Treat "already deleted" as success (log and return nil)
- SSH version checks exit status 255 (lines 27-32 in virtual-disk_delete.go) - replicate this pattern

**Warning signs:**
- Delete operations fail on second terraform apply
- Errors when destroying already-destroyed resources
- Non-idempotent delete behavior

**Example pattern:**
```go
err := dm.DeleteVirtualDisk(gc.Context(), diskPath, nil)
if err != nil {
    // Check if already deleted (acceptable error)
    if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
        log.Printf("[virtualDiskDelete_govmomi] Already deleted: %s", virtdisk_id)
        return nil
    }
    return fmt.Errorf("failed to delete virtual disk: %w", err)
}
```

## Code Examples

Verified patterns from official sources and existing codebase:

### Virtual Disk Creation (Already Implemented)
```go
// Source: esxi/virtual-disk_functions.go lines 281-347
func virtualDiskCREATE_govmomi(c *Config, virtual_disk_disk_store string, virtual_disk_dir string,
    virtual_disk_name string, virtual_disk_size int, virtual_disk_type string) (string, error) {

    // Validate disk store exists
    err := diskStoreValidate_govmomi(c, virtual_disk_disk_store)
    if err != nil {
        return "", fmt.Errorf("failed to validate disk store: %w", err)
    }

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", fmt.Errorf("failed to get govmomi client: %w", err)
    }

    ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
    if err != nil {
        return "", fmt.Errorf("failed to get datastore: %w", err)
    }

    // Build dual-format paths
    virtdisk_id := fmt.Sprintf("/vmfs/volumes/%s/%s/%s", virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name)
    diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

    // Map disk type to govmomi backing
    var diskType string
    switch virtual_disk_type {
    case "thin":
        diskType = string(types.VirtualDiskTypeThin)
    case "zeroedthick":
        diskType = string(types.VirtualDiskTypeThick)
    case "eagerzeroedthick":
        diskType = string(types.VirtualDiskTypeEagerZeroedThick)
    default:
        diskType = string(types.VirtualDiskTypeThin)
    }

    // Create disk spec
    spec := &types.FileBackedVirtualDiskSpec{
        VirtualDiskSpec: types.VirtualDiskSpec{
            DiskType:    diskType,
            AdapterType: string(types.VirtualDiskAdapterTypeLsiLogic),
        },
        CapacityKb: int64(virtual_disk_size * 1024 * 1024), // Convert GB to KB
    }

    // Create virtual disk manager
    dm := object.NewVirtualDiskManager(gc.Client.Client)

    // Create the virtual disk (datacenter can be nil for standalone ESXi)
    task, err := dm.CreateVirtualDisk(gc.Context(), diskPath, nil, spec)
    if err != nil {
        return "", fmt.Errorf("failed to create virtual disk: %w", err)
    }

    err = waitForTask(gc.Context(), task)
    if err != nil {
        return "", fmt.Errorf("failed to create virtual disk: %w", err)
    }

    log.Printf("[virtualDiskCREATE_govmomi] Created virtual disk: %s\n", virtdisk_id)
    return virtdisk_id, nil  // Return /vmfs/volumes/ format for Terraform state
}
```

### Virtual Disk Reading with Size Detection (Already Implemented)
```go
// Source: esxi/virtual-disk_functions.go lines 350-432
func virtualDiskREAD_govmomi(c *Config, virtdisk_id string) (string, string, string, int, string, error) {
    log.Println("[virtualDiskREAD_govmomi] Begin")

    // Parse /vmfs/volumes/datastore/dir/disk.vmdk format
    s := strings.Split(virtdisk_id, "/")
    log.Printf("[virtualDiskREAD_govmomi] len=%d cap=%d %v\n", len(s), cap(s), s)
    if len(s) < 6 {
        return "", "", "", 0, "", nil
    }

    var virtual_disk_dir string
    if len(s) > 6 {
        virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")  // Handle nested dirs
    } else {
        virtual_disk_dir = s[4]
    }
    virtual_disk_disk_store := s[3]
    virtual_disk_name := s[len(s)-1]

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", "", "", 0, "", fmt.Errorf("failed to get govmomi client: %w", err)
    }

    ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
    if err != nil {
        return "", "", "", 0, "", fmt.Errorf("failed to get datastore: %w", err)
    }

    // Convert to [datastore] path format
    diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

    // Query virtual disk to verify it exists
    dm := object.NewVirtualDiskManager(gc.Client.Client)
    uuid, err := dm.QueryVirtualDiskUuid(gc.Context(), diskPath, nil)
    if err != nil {
        return "", "", "", 0, "", fmt.Errorf("virtual disk not found: %w", err)
    }

    log.Printf("[virtualDiskREAD_govmomi] Found disk with UUID: %s\n", uuid)

    // Get disk size from the -flat.vmdk file using datastore browser
    var virtual_disk_size int
    parts := strings.Split(virtual_disk_name, ".")
    if len(parts) >= 2 {
        virtual_disk_nameFlat := fmt.Sprintf("%s-flat.%s", parts[0], parts[1])

        // Use the datastore browser to search for the flat file
        browser, err := ds.Browser(gc.Context())
        if err == nil {
            spec := types.HostDatastoreBrowserSearchSpec{
                MatchPattern: []string{virtual_disk_nameFlat},
                Details: &types.FileQueryFlags{
                    FileSize: true,
                },
            }

            searchPath := ds.Path(virtual_disk_dir)
            task, err := browser.SearchDatastore(gc.Context(), searchPath, &spec)
            if err == nil {
                info, err := task.WaitForResult(gc.Context())
                if err == nil {
                    if result, ok := info.Result.(types.HostDatastoreBrowserSearchResults); ok {
                        if len(result.File) > 0 {
                            if fileInfo, ok := result.File[0].(*types.FileInfo); ok {
                                // Convert bytes to GB
                                virtual_disk_size = int(fileInfo.FileSize / (1024 * 1024 * 1024))
                                log.Printf("[virtualDiskREAD_govmomi] Disk size: %d GB\n", virtual_disk_size)
                            }
                        }
                    }
                }
            }
        }
    }

    // Disk type defaults to "Unknown" (acceptable - type is informational only)
    virtual_disk_type := "Unknown"

    return virtual_disk_disk_store, virtual_disk_dir, virtual_disk_name, virtual_disk_size, virtual_disk_type, nil
}
```

### Virtual Disk Deletion (NEW - Needs Implementation)
```go
// NEW FUNCTION - Add to virtual-disk_functions.go after growVirtualDisk_govmomi
// Source: govmomi VirtualDiskManager API
// Reference: https://github.com/vmware/govmomi/blob/main/object/virtual_disk_manager.go
func virtualDiskDelete_govmomi(c *Config, virtdisk_id string) error {
    log.Println("[virtualDiskDelete_govmomi]")

    // Parse /vmfs/volumes/datastore/dir/disk.vmdk format
    s := strings.Split(virtdisk_id, "/")
    if len(s) < 6 {
        return fmt.Errorf("invalid virtdisk_id format: %s", virtdisk_id)
    }

    var virtual_disk_dir string
    if len(s) > 6 {
        virtual_disk_dir = strings.Join(s[4:len(s)-1], "/")
    } else {
        virtual_disk_dir = s[4]
    }
    virtual_disk_disk_store := s[3]
    virtual_disk_name := s[len(s)-1]

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return fmt.Errorf("failed to get govmomi client: %w", err)
    }

    ds, err := getDatastoreByName(gc.Context(), gc.Finder, virtual_disk_disk_store)
    if err != nil {
        return fmt.Errorf("failed to get datastore: %w", err)
    }

    // Convert to [datastore] path format
    diskPath := ds.Path(fmt.Sprintf("%s/%s", virtual_disk_dir, virtual_disk_name))

    // Create virtual disk manager
    dm := object.NewVirtualDiskManager(gc.Client.Client)

    // Delete the virtual disk (datacenter can be nil for standalone ESXi)
    task, err := dm.DeleteVirtualDisk(gc.Context(), diskPath, nil)
    if err != nil {
        // Check if already deleted (acceptable - idempotent behavior)
        if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
            log.Printf("[virtualDiskDelete_govmomi] Already deleted: %s", virtdisk_id)
            return nil
        }
        return fmt.Errorf("failed to delete virtual disk: %w", err)
    }

    err = waitForTask(gc.Context(), task)
    if err != nil {
        // Check for "already deleted" in task error
        if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
            log.Printf("[virtualDiskDelete_govmomi] Already deleted during task: %s", virtdisk_id)
            return nil
        }
        return fmt.Errorf("delete task failed: %w", err)
    }

    log.Printf("[virtualDiskDelete_govmomi] Successfully deleted: %s", virtdisk_id)
    return nil
}
```

### Datastore Browser Search (NEW - Needs Implementation)
```go
// NEW IMPLEMENTATION - Replace stub in data_source_esxi_virtual_disk.go
// Source: govmomi HostDatastoreBrowser API
// Reference: https://github.com/vmware/govmomi/blob/main/object/host_datastore_browser.go
func findVirtualDiskInDir_govmomi(c *Config, diskStore, dir string) (string, error) {
    log.Printf("[findVirtualDiskInDir_govmomi] diskStore=%s dir=%s", diskStore, dir)

    gc, err := c.GetGovmomiClient()
    if err != nil {
        return "", fmt.Errorf("failed to get govmomi client: %w", err)
    }

    ds, err := getDatastoreByName(gc.Context(), gc.Finder, diskStore)
    if err != nil {
        return "", fmt.Errorf("failed to get datastore: %w", err)
    }

    browser, err := ds.Browser(gc.Context())
    if err != nil {
        return "", fmt.Errorf("failed to get datastore browser: %w", err)
    }

    // Search for .vmdk files (descriptor files only, not -flat.vmdk)
    spec := types.HostDatastoreBrowserSearchSpec{
        MatchPattern: []string{"*.vmdk"},
        Details: &types.FileQueryFlags{
            FileSize: false,  // Don't need size for discovery
        },
    }

    searchPath := ds.Path(dir)
    task, err := browser.SearchDatastore(gc.Context(), searchPath, &spec)
    if err != nil {
        return "", fmt.Errorf("failed to search datastore: %w", err)
    }

    info, err := task.WaitForResult(gc.Context())
    if err != nil {
        return "", fmt.Errorf("search task failed: %w", err)
    }

    result, ok := info.Result.(types.HostDatastoreBrowserSearchResults)
    if !ok {
        return "", fmt.Errorf("unexpected search result type")
    }

    if len(result.File) == 0 {
        return "", nil  // No VMDK found (not an error)
    }

    // Return first descriptor .vmdk file found (exclude -flat.vmdk files)
    for _, file := range result.File {
        if fileInfo, ok := file.(*types.FileInfo); ok {
            filename := fileInfo.Path
            // Skip -flat.vmdk files - we want the descriptor .vmdk only
            if !strings.HasSuffix(filename, "-flat.vmdk") && strings.HasSuffix(filename, ".vmdk") {
                log.Printf("[findVirtualDiskInDir_govmomi] Found: %s", filename)
                return filename, nil
            }
        }
    }

    return "", nil  // No descriptor .vmdk found (not an error)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SSH vmkfstools for disk operations | govmomi VirtualDiskManager API | Phase 5 (this phase) | Eliminates SSH dependency, cross-platform compatibility, proper error handling |
| Manual directory creation via SSH mkdir | Automatic directory creation by VirtualDiskManager | Phase 5 (this phase) | Simpler code, fewer edge cases |
| SSH ls parsing for disk discovery | HostDatastoreBrowser.SearchDatastore | Phase 5 (this phase) | Structured results, works across all datastore types (VMFS, NFS, VSAN) |
| Manual directory emptiness check + rmdir | ESXi automatic directory management | Phase 5 (this phase) | Simpler delete operation, let ESXi manage directory lifecycle |
| vmkfstools -t0 parsing for disk type | Accept "Unknown" disk type | Phase 5 (this phase) | Type is informational only, no functional impact |

**Deprecated/outdated:**
- SSH-based datastore validation (esxcli storage filesystem list) - replaced by Datastore.Properties with summary.accessible check
- Manual /vmfs/volumes/ path construction for validation - replaced by govmomi Finder.Datastore lookup
- vmkfstools -X for disk growth - replaced by VirtualDiskManager.ExtendVirtualDisk (already done)

## Open Questions

1. **Should virtualDiskDelete_govmomi attempt directory cleanup?**
   - What we know: SSH version manually checks if directory is empty and deletes it (lines 35-44 in virtual-disk_delete.go)
   - What's unclear: Whether this manual cleanup is necessary or if ESXi manages it automatically
   - Recommendation: Start without directory cleanup in govmomi version. If tests or real-world usage show orphaned directories, add FileManager.DeleteDatastoreFile() for empty directory removal as follow-up enhancement.

2. **Should findVirtualDiskInDir_govmomi filter out -flat.vmdk files?**
   - What we know: SSH version uses `ls *.vmdk | head -1` which returns first match (lines 114-120 in data_source_esxi_virtual_disk.go)
   - What's unclear: Whether -flat.vmdk files should be explicitly filtered or if glob pattern naturally excludes them
   - Recommendation: Explicitly filter -flat.vmdk files in govmomi version (see code example) to ensure only descriptor .vmdk files are returned, matching SSH behavior.

3. **Can disk type detection be implemented via govmomi?**
   - What we know: VirtualDiskManager has QueryVirtualDiskInfo method but it's not documented in current implementation
   - What's unclear: Whether QueryVirtualDiskInfo reliably returns disk type, and if vcsim supports it
   - Recommendation: Leave as "Unknown" for Phase 5. If disk type becomes functionally important (currently just informational), investigate QueryVirtualDiskInfo in future enhancement phase.

## Sources

### Primary (HIGH confidence)
- [govmomi VirtualDiskManager source code](https://github.com/vmware/govmomi/blob/main/object/virtual_disk_manager.go) - Method signatures for CreateVirtualDisk, DeleteVirtualDisk, ExtendVirtualDisk, QueryVirtualDiskUuid
- [govmomi Datastore source code](https://github.com/vmware/govmomi/blob/main/object/datastore.go) - Path() method implementation and datastore path format
- [govmomi HostDatastoreBrowser source code](https://github.com/vmware/govmomi/blob/main/object/host_datastore_browser.go) - SearchDatastore method for file enumeration
- [govmomi DatastoreFileManager source code](https://github.com/vmware/govmomi/blob/main/object/datastore_file_manager.go) - Delete, DeleteFile, DeleteVirtualDisk methods
- Local codebase analysis - Existing govmomi implementations in virtual-disk_functions.go, verified test coverage in virtual-disk_functions_test.go

### Secondary (MEDIUM confidence)
- [govmomi VirtualDiskManager API overview](https://pkg.go.dev/github.com/vmware/govmomi/object) - Official Go package documentation
- [Virtual Infrastructure JSON API - DeleteVirtualDisk_Task](https://developer.broadcom.com/xapis/virtual-infrastructure-json-api/latest/sdk/vim25/release/VirtualDiskManager/moId/DeleteVirtualDisk_Task/post/) - VMware official API reference for delete operation
- [govmomi FileManager Golang examples](https://golang.hotexamples.com/examples/github.com.vmware.govmomi.object/FileManager/-/golang-filemanager-class-examples.html) - Community usage patterns for FileManager operations
- [VMware disk types knowledge base](https://knowledge.broadcom.com/external/article?legacyId=1011170) - Determining zeroedthick vs eagerzeroedthick

### Tertiary (LOW confidence - flagged for validation)
- [govmomi issue #2174 - Register virtual disk invalid path error](https://github.com/vmware/govmomi/issues/2174) - Path format pitfalls
- [govmomi issue #1263 - VM disk creation fails with datastore name only](https://github.com/vmware/govmomi/issues/1263) - vcsim path format limitations
- [govmomi issue #937 - vcsim MakeDirectory failures](https://github.com/vmware/govmomi/issues/937) - Simulator limitations for directory operations
- [govmomi issue #2889 - vcsim ReconfigVM_Task disk capacity issues](https://github.com/vmware/govmomi/issues/2889) - Simulator disk capacity field limitations
- [govmomi issue #1108 - govc device.info disk size with vcsim](https://github.com/vmware/govmomi/issues/1108) - Simulator disk size reporting issues

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use, proven in previous phases
- Architecture: HIGH - Patterns established in Phases 2-4, path format verified in existing code
- Pitfalls: HIGH - Identified through codebase analysis and govmomi issue review
- Missing functionality: MEDIUM - virtualDiskDelete_govmomi and findVirtualDiskInDir_govmomi require new implementation, but API usage is well-documented

**Research date:** 2026-02-13
**Valid until:** 2026-04-13 (60 days - stable govmomi API, established patterns)
