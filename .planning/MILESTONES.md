# Milestones

## v1.0 Build Fix & SSH Removal (Shipped: 2026-02-14)

**Phases:** 1-6 | **Plans:** 7 | **Tasks:** 15 | **Timeline:** 2 days (2026-02-13 - 2026-02-14)
**Files modified:** 58 | **Lines:** +7,149 / -1,683 | **Codebase:** 8,601 LOC Go
**Git range:** eb0e224..00dfb1a

**Delivered:** Provider compiles cleanly with govmomi-first architecture — SSH removed from portgroup, vswitch, resource pool, and virtual disk; retained only for guest VM operations.

**Key accomplishments:**
1. Fixed build errors and implemented govmomi-based esxi_host data source with full hardware/datastore enumeration
2. Removed SSH from portgroup resource — established removal pattern for remaining resources
3. Removed SSH from vSwitch resource — completed network resource cleanup
4. Removed SSH from resource pool resource — govmomi-only compute resource operations
5. Removed SSH from virtual disk — implemented VirtualDiskManager delete and datastore browser
6. Eliminated useGovmomi feature flag, renamed all functions, removed 620 lines of dead code

---

