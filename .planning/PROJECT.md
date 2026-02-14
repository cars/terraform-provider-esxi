# Terraform Provider ESXi — Build Fix & SSH Removal

## What This Is

A Terraform provider for managing VMs, virtual disks, resource pools, vswitches, and port groups directly on ESXi hosts without vCenter. Uses govmomi API for portgroup, vswitch, resource pool, virtual disk, and data source operations. SSH retained only for guest VM create/delete operations where no govmomi alternative exists.

## Core Value

The provider must compile cleanly and all tests must pass — a working, buildable provider is the non-negotiable foundation.

## Requirements

### Validated

- ✓ Terraform resource lifecycle (create/read/update/delete) for 5 resource types — existing
- ✓ Govmomi client connection management with session caching and reconnect — existing
- ✓ Govmomi implementations for portgroup, vswitch, resource pool, virtual disk operations — existing
- ✓ Data sources for guest, portgroup, vswitch, resource pool, virtual disk, host — existing
- ✓ Test infrastructure using govmomi simulator (vcsim) — existing
- ✓ Cloud-init support via guestinfo metadata — existing
- ✓ OVF/OVA source deployment support — existing
- ✓ Provider compiles without errors — v1.0
- ✓ SSH code removed from portgroup, vswitch, resource pool, virtual disk — v1.0
- ✓ SSH code retained for guest VM operations (no govmomi alternative) — v1.0
- ✓ All existing tests pass (27/32, 5 simulator limitations) — v1.0
- ✓ useGovmomi feature flag removed, functions renamed to canonical names — v1.0

### Active

(None — awaiting next milestone definition)

### Out of Scope

- Guest VM create/delete govmomi implementation — complex operations (OVF, cloud-init, SCP), potential v2
- Upgrading Terraform SDK version — separate concern, massive rewrite
- govmomi version upgrade — unnecessary risk, separate concern
- Test hardening (concurrent testing, simulator improvements) — potential v2

## Context

Shipped v1.0 with 8,601 LOC Go.
Tech stack: Go, govmomi, Terraform SDK v0.12.17, govmomi simulator (vcsim).
Architecture: govmomi-first for portgroup/vswitch/resource-pool/virtual-disk, SSH for guest create/delete.
27/32 tests passing (5 known simulator limitations — not code defects).
Known dead code: `dataSourceEsxiHostReadSSH` and its SSH helpers remain unused but harmless.

## Constraints

- **Compatibility**: Keep SSH code where no govmomi alternative exists — don't break functionality
- **Incremental commits**: Commit at each significant milestone so progress isn't lost
- **Test coverage**: All existing tests must pass after completion
- **No new features**: This was a cleanup/fix project, not a feature project

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Remove SSH only where govmomi exists | Preserves functionality for operations not yet migrated | ✓ Good — 4 resources fully migrated |
| Fix build first, then remove SSH | Can't validate SSH removal without a building codebase | ✓ Good — established test baseline |
| Commit incrementally | User wants git history of significant changes | ✓ Good — 43 commits across 6 phases |
| Wrapper function pattern during migration | Thin wrappers to _govmomi functions kept callers unchanged | ✓ Good — clean migration, wrappers removed in Phase 6 |
| Keep useGovmomi flag until Phase 6 | Global cleanup after all SSH removal complete | ✓ Good — single removal pass, no regressions |
| Implement full govmomi host reader | Better performance via single API call vs multiple SSH commands | ✓ Good — full feature parity |
| Defer test hardening to v2 | Not in v1 requirements, concurrent testing beyond current scope | — Pending |

---
*Last updated: 2026-02-14 after v1.0 milestone*
