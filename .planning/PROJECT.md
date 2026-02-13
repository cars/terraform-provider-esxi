# Terraform Provider ESXi — Build Fix & SSH Removal

## What This Is

A Terraform provider for managing VMs, virtual disks, resource pools, vswitches, and port groups directly on ESXi hosts without vCenter. Currently in a dual-path state (SSH + govmomi) with build errors. This project fixes the build and completes the migration to govmomi by removing SSH code wherever a govmomi alternative exists.

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

### Active

- [ ] Provider compiles without errors (fix data_source_esxi_host.go build failures)
- [ ] SSH code removed from all resources that have govmomi alternatives
- [ ] SSH code retained only where no govmomi implementation exists
- [ ] All existing tests pass after changes
- [ ] Significant changes committed incrementally to git

### Out of Scope

- Adding new resources or data sources — focus is on fixing and cleaning existing code
- Upgrading Terraform SDK version — separate concern, don't mix with this work
- Adding new govmomi implementations where none exist — only remove SSH where govmomi already covers it
- Refactoring govmomi code — leave working govmomi implementations as-is

## Context

- Build is broken: all errors in `esxi/data_source_esxi_host.go` — missing `dataSourceEsxiHostReadGovmomi` function and type mismatches between `ConnectionStruct` and `map[string]string`
- Provider uses dual-path routing via `Config.useGovmomi` flag
- Govmomi path uses `govmomi_client.go` (connection/session) and `govmomi_helpers.go` (shared utilities)
- SSH path uses `esxi_remote_cmds.go` to execute vim-cmd, esxcli over SSH
- Each resource follows pattern: `resource_<type>.go` (schema), `<type>-create.go`, `<type>-read.go`, etc.
- Tests use `simulator.ESX()` for standalone ESXi simulation — no real hardware needed
- Codebase map available at `.planning/codebase/`

## Constraints

- **Compatibility**: Keep SSH code where no govmomi alternative exists — don't break functionality
- **Incremental commits**: Commit at each significant milestone so progress isn't lost
- **Test coverage**: All existing tests must pass after completion
- **No new features**: This is a cleanup/fix project, not a feature project

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Remove SSH only where govmomi exists | Preserves functionality for operations not yet migrated | — Pending |
| Fix build first, then remove SSH | Can't validate SSH removal without a building codebase | — Pending |
| Commit incrementally | User wants git history of significant changes | — Pending |

---
*Last updated: 2026-02-12 after initialization*
