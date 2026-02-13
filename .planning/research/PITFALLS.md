# Domain Pitfalls: Terraform Provider SSH-to-govmomi Migration

**Domain:** Terraform provider migration from SSH commands to govmomi API
**Researched:** 2026-02-12
**Overall confidence:** HIGH

## Critical Pitfalls

Mistakes that cause rewrites, data loss, or major production issues.

### Pitfall 1: Dual-Path State Inconsistency During Migration
**What goes wrong:** Resources created with SSH cannot be properly read/updated by govmomi code, or vice versa. State file shows resource exists but govmomi cannot find it, leading to "resource not found" errors and attempted recreation of existing resources.

**Why it happens:**
- SSH commands and govmomi API may return different identifiers for the same resource
- SSH uses vim-cmd which returns vmid, govmomi uses ManagedObjectReference
- Resource pool paths differ between vim-cmd and govmomi API
- Virtual disk identifiers formatted differently ([datastore] path vs datastore:/path)

**Consequences:**
- Terraform attempts to recreate existing VMs/resources
- State drift that cannot be resolved without manual intervention
- Data loss if Terraform destroys and recreates resources
- Users blocked from upgrading provider version

**Prevention:**
1. Ensure ID format is canonical across both implementations (use MOIDs or convert consistently)
2. Add migration logic in Read operations to handle legacy IDs
3. Test import functionality with resources created by SSH path
4. Never remove SSH path until ALL production users have migrated state
5. Document breaking changes and provide migration guide

**Detection:**
- Test suite shows "resource not found" on reads after switching useGovmomi flag
- Import fails for SSH-created resources when useGovmomi=true
- State refresh shows resources need recreation
- Users report "resource already exists" errors on apply

**Phase mapping:** Phase 1 (Fix Build) must establish ID format standards before Phase 2 (Remove SSH)

---

### Pitfall 2: Session Management and Resource Cleanup
**What goes wrong:** govmomi sessions leak or expire mid-operation. Operations fail with "session not authenticated" errors. Provider holds connections open indefinitely, exhausting ESXi connection limits.

**Why it happens:**
- Current code caches GovmomiClient in Config struct but may not properly clean up
- Context cancellation not consistently propagated through operation chains
- No session keepalive for long-running operations
- defer Close() in tests but not in actual resource operations
- Terraform can run many operations in parallel, each potentially creating new sessions

**Consequences:**
- ESXi host refuses new connections (max 32 concurrent sessions on free ESXi)
- Mid-operation failures with cryptic session errors
- Orphaned resources when session expires during creation
- Memory leaks in provider process
- Cannot run terraform plan/apply on large infrastructure

**Prevention:**
1. Implement session keepalive for operations >30 minutes
2. Add session validation before each operation (reconnect if expired)
3. Ensure client Close() called via defer in resource CRUD operations
4. Use connection pooling with proper lifecycle management
5. Add context timeout to all govmomi operations (currently 30min in waitForTask)
6. Test with parallel resource operations to verify session limits

**Detection:**
- Error: "session is not authenticated"
- ESXi host log shows session limit exceeded
- Provider process memory grows over time
- netstat shows many CLOSE_WAIT connections to ESXi
- Operations hang without timeout

**Phase mapping:** Phase 1 must fix session management before removing SSH fallback

---

### Pitfall 3: Error Handling Differences Between SSH and govmomi
**What goes wrong:** SSH commands return success (exit 0) even when operation partially failed. govmomi returns typed errors that need different handling. Resource appears created but is actually broken.

**Why it happens:**
- SSH path uses stdout parsing and regex matching (fragile)
- vim-cmd/esxcli sometimes return 0 even on errors
- govmomi returns proper fault types but code doesn't check them
- SSH error detection relies on string matching ("Error:", "Failed")
- Race conditions: SSH command returns before operation completes, govmomi waits for tasks

**Consequences:**
- Silent failures create inconsistent state
- Resources marked as created but don't actually exist
- Update operations appear successful but don't apply changes
- Error messages unhelpful ("command failed" vs specific fault)
- Terraform state diverges from actual ESXi state

**Prevention:**
1. Add comprehensive fault type checking for all govmomi operations
2. Never assume task completion without WaitForTask
3. Verify resource actually exists after creation (read-after-write)
4. Test error cases explicitly (insufficient resources, invalid config, etc.)
5. Map govmomi faults to clear user messages
6. Add validation before operations (e.g., diskStoreValidate_govmomi)

**Detection:**
- Resources in state file don't exist on ESXi host
- "terraform plan" shows no changes but manual check shows drift
- Errors lack specific details
- Test code has fewer error assertions for govmomi path
- grep for "if err != nil" shows govmomi functions with weak error handling

**Phase mapping:** Phase 1 must harden error handling before removing SSH safety net

---

### Pitfall 4: Feature Parity Gaps (govmomi Cannot Do Everything SSH Can)
**What goes wrong:** Certain operations possible via esxcli/vim-cmd have no govmomi API equivalent. Removing SSH eliminates features users depend on.

**Why it happens:**
- govmomi wraps vSphere API, which doesn't expose all ESXi-specific features
- SSH commands can access ESXi CLI tools not available via API
- Some configurations require esxcli (advanced networking, storage)
- Undocumented vim-cmd features used by provider
- vSwitch uplink management more flexible via esxcli

**Consequences:**
- Feature regression when removing SSH
- Cannot implement certain resource updates
- Users blocked from managing some configurations
- Provider becomes less useful than SSH version

**Prevention:**
1. Audit ALL esxcli/vim-cmd commands used in SSH path
2. Document which operations have no govmomi equivalent BEFORE removing SSH
3. Keep SSH path for operations without govmomi alternative
4. Add warnings in documentation for features at risk
5. Test matrix: verify every SSH operation has govmomi test coverage

**Detection:**
- SSH functions that have no corresponding _govmomi function
- grep shows esxcli/vim-cmd commands with no govmomi alternative
- Code has "if c.useGovmomi" with SSH-only operations in else block
- No govmomi test for specific feature that SSH tests cover

**Current gaps found in codebase:**
- esxcli storage filesystem rescan (virtual-disk_functions.go:41)
- Some vswitch uplink operations may be limited
- OVF deployment via ovftool (guest-create.go uses HTTP/ovftool, not govmomi)

**Phase mapping:** Phase 1 must complete feature parity audit before Phase 2 removal

---

### Pitfall 5: Concurrent Operation Safety (govmomi vs SSH)
**What goes wrong:** govmomi operations acquire locks on ESXi objects. Concurrent terraform operations deadlock or corrupt state. SSH commands don't coordinate, leading to race conditions.

**Why it happens:**
- vSphere API uses object locking for consistency
- Terraform parallelizes independent resource operations
- Multiple resources may modify same underlying object (vswitch with portgroups)
- SSH commands are stateless, govmomi maintains session/lock state
- ReconfigVM operations lock VM until task completes

**Consequences:**
- Terraform hangs waiting for locks
- Operations fail with "object is busy"
- VM configuration corruption from concurrent updates
- Cannot create multiple resources simultaneously
- Timeouts in large infrastructure deployments

**Prevention:**
1. Document resource dependencies (portgroup requires vswitch)
2. Use proper task waiting (WaitForTask) before releasing
3. Add retry logic with exponential backoff for lock conflicts
4. Test parallel resource creation/updates
5. Consider terraform depends_on hints in documentation

**Detection:**
- Operations timeout after 30 minutes
- Error: "object is busy" or "another task is already in progress"
- terraform apply hangs with no progress
- Test suite has race conditions (run with -race flag)
- Users report "works sequentially but fails in parallel"

**Phase mapping:** Phase 1 should add concurrency tests before widespread adoption

---

## Moderate Pitfalls

Cause frustration, workarounds needed, but not catastrophic.

### Pitfall 6: IP Address Detection Timing Differences
**What goes wrong:** govmomi's waitForGuestIPAddress uses VMware Tools property.Wait(), SSH uses vim-cmd polling. Different timing and reliability.

**Why it happens:**
- property.Wait() is event-driven, SSH is polling
- govmomi waits for guest.ipAddress property, SSH parses output
- VMware Tools must be running for both, but report differently
- Timeout handling differs (30min vs guest_startup_timeout)

**Consequences:**
- Guest creation succeeds with SSH but times out with govmomi
- IP address reported differs (first interface vs specific NIC)
- Longer waits with govmomi if Tools not installed

**Prevention:**
- Use same timeout value (guest_startup_timeout) in both paths
- Document VMware Tools requirement prominently
- Add fallback to manual IP lookup if property.Wait() times out
- Test with guests that have no Tools installed

**Phase mapping:** Phase 1 - ensure timeouts consistent before removing SSH

---

### Pitfall 7: Virtual Disk Path Format Inconsistency
**What goes wrong:** SSH path uses `[datastore] dir/disk.vmdk`, govmomi may use different format. virtdisk_id incompatible between implementations.

**Why it happens:**
- Different parsing of esxcli storage output vs datastore browser
- Virtual disk ID construction differs
- govmomi uses Datastore object references, SSH uses string paths

**Consequences:**
- Virtual disk import fails
- Cannot grow disks created via SSH path
- State shows wrong disk paths

**Prevention:**
- Standardize on datastore path format: `[datastore] dir/disk.vmdk`
- Add path normalization function used by both implementations
- Test import with SSH-created disks

**Phase mapping:** Phase 1 - add path normalization before Phase 2

---

### Pitfall 8: Security Policy Behavior Differences (vSwitch/Portgroup)
**What goes wrong:** esxcli and govmomi handle security policy inheritance differently. Setting promiscuous mode via SSH affects different scope than govmomi.

**Why it happens:**
- esxcli sets policy at vswitch OR portgroup level
- govmomi API has explicit inheritance flags
- Default values differ between methods

**Consequences:**
- Security policy not applied as expected
- Promiscuous mode enabled when user expected disabled
- Cannot match exact SSH behavior with govmomi

**Prevention:**
- Read current policy before updates
- Explicitly set all policy fields (no relying on defaults)
- Test policy inheritance scenarios
- Document breaking changes if behavior must differ

**Phase mapping:** Phase 1 - verify security policy parity

---

### Pitfall 9: Resource Pool Path Resolution
**What goes wrong:** SSH uses resource pool names, govmomi uses ManagedObjectReference IDs. Path traversal differs.

**Why it happens:**
- getPoolID_govmomi vs SSH path use different lookup mechanisms
- Nested resource pools require path traversal in govmomi
- ManagedObjectReference format: resgroup-XXX vs human-readable name

**Consequences:**
- Cannot find resource pool by name
- Nested pools not resolved correctly
- Import fails for pools created via SSH

**Prevention:**
- Implement robust findResourcePoolByPath in govmomi_helpers.go
- Support both path and MOID for lookups
- Test deeply nested resource pools
- Handle root pool ("Resources") specially

**Phase mapping:** Phase 1 - verify pool resolution before SSH removal

---

## Minor Pitfalls

Annoyances that are easy to fix.

### Pitfall 10: Test Coverage Gaps for govmomi Path
**What goes wrong:** Tests exist but only cover SSH path. govmomi code not actually tested.

**Why it happens:**
- Tests written before govmomi implementation
- useGovmomi flag not set in older tests
- Simulator tests added later

**Consequences:**
- False confidence in govmomi implementation
- Bugs found in production, not CI
- Regressions when refactoring

**Prevention:**
- Every test should have govmomi variant with useGovmomi: true
- Current tests properly use simulator.ESX() and set useGovmomi
- Add test to verify both paths return identical results

**Phase mapping:** Phase 1 - achieve 100% test coverage before SSH removal

---

### Pitfall 11: Context Propagation Inconsistency
**What goes wrong:** Some govmomi functions create new context, others use client context. Timeout behavior unpredictable.

**Why it happens:**
- waitForTask creates context.WithTimeout(ctx, 30*time.Minute)
- Some functions use gc.ctx from client
- No consistent context passing pattern

**Consequences:**
- Cannot cancel long-running operations
- Timeouts not respected
- Memory leaks from uncancelled contexts

**Prevention:**
- Pass context as first parameter to all functions (Go convention)
- Use client context as parent for operation-specific contexts
- Always pair WithTimeout with defer cancel()

**Phase mapping:** Phase 1 - standardize context usage

---

### Pitfall 12: Provider Configuration Migration (useGovmomi Flag Confusion)
**What goes wrong:** Users don't understand when to enable useGovmomi flag. Mixed results if some resources use SSH, others use govmomi.

**Why it happens:**
- Feature flag is implementation detail exposed to users
- No migration guide
- Default value unclear

**Consequences:**
- User confusion
- Support burden
- Inconsistent behavior across resources

**Prevention:**
- Make govmomi the default when feature-complete
- Deprecate useGovmomi flag (use environment variable for testing)
- Provide migration guide for state refresh

**Phase mapping:** Phase 2 - remove flag once SSH code deleted

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Fix Build (Phase 1) | Introducing new bugs while fixing compilation | Add comprehensive test suite FIRST, then fix build |
| Fix Build | Breaking existing SSH functionality | Do not modify SSH code, only fix imports/types |
| Remove SSH Code (Phase 2) | Removing code still used by production | Check ALL useGovmomi branches, verify govmomi path exists |
| Remove SSH Code | State migration breaks existing users | Provide state migration tool or document manual steps |
| Remove SSH Code | Deleting esxi_remote_cmds.go too early | Keep file until every vim-cmd/esxcli call replaced |
| Cleanup | Removing Config.govmomiClient cache breaks performance | Verify session reuse works before removing optimization |
| Cleanup | Deleting GovmomiClient.Close() causes leaks | Ensure all callers migrated to new pattern first |

---

## Pre-Removal Checklist

Before removing SSH code, verify:

- [ ] Every `if c.useGovmomi` branch has working govmomi implementation
- [ ] All govmomi functions have test coverage with simulator
- [ ] ID formats compatible between SSH and govmomi (or migration documented)
- [ ] Session management tested under load (parallel operations)
- [ ] Error handling validates actual resource state, not just API success
- [ ] Feature parity audit completed (document any gaps)
- [ ] State migration guide written
- [ ] Import tested for SSH-created resources using govmomi path
- [ ] No references to vim-cmd/esxcli in govmomi functions
- [ ] Context handling consistent across all operations
- [ ] Connection cleanup tested (no leaks)

---

## Research Methodology

**Sources analyzed:**
1. Codebase examination: 17 files with `if c.useGovmomi` branches
2. Test coverage analysis: 8 test files using govmomi simulator
3. SSH command inventory: 49 vim-cmd/esxcli invocations across 19 files
4. Error handling patterns: govmomi_helpers.go, *_functions.go files
5. Session management: govmomi_client.go, config.go
6. Training data: Terraform provider development patterns (MEDIUM confidence)
7. Training data: govmomi API limitations vs ESXi CLI (MEDIUM confidence)

**Confidence levels:**
- Session management pitfalls: HIGH (observed in code)
- State inconsistency risks: HIGH (observed dual-path implementation)
- Feature parity gaps: MEDIUM (found some, may be more)
- govmomi API limitations: MEDIUM (based on training data, not exhaustive API review)
- Error handling differences: HIGH (observed pattern differences)

**Limitations:**
- Could not verify govmomi API documentation via Context7 (library not available)
- No web search access to find community experience with similar migrations
- Analysis based on codebase state and training data only
- Some feature gaps may exist that aren't apparent from code inspection

**What might be missed:**
- Undocumented ESXi API behaviors
- Version-specific govmomi compatibility issues
- Performance differences under production load
- Edge cases in specific ESXi configurations (vSAN, distributed switches, etc.)
- VMware Tools version compatibility matrix
