# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for VMware ESXi — manages VMs, virtual disks, resource pools, vswitches, and port groups directly on ESXi hosts without requiring vCenter. All source code is in the `esxi/` package.

## Build & Test Commands

```bash
# Build (static Linux binary)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o terraform-provider-esxi_$(cat version)

# Run all tests
go test ./esxi/ -v

# Run a single test
go test ./esxi/ -v -run TestPortgroupCreateReadDeleteGovmomi

# Build check (compile only)
go build ./...
```

The version string comes from the `version` file in the repo root.

## Architecture

### Dual-Path Implementation (SSH + govmomi)

The provider is undergoing a phased migration from SSH-based ESXi commands to the govmomi API. Each resource's functions check `Config.useGovmomi` to route to the appropriate implementation:

```go
func someOperation(c *Config, ...) {
    if c.useGovmomi {
        return someOperation_govmomi(c, ...)
    }
    // SSH fallback via runRemoteSshCommand()
}
```

- **SSH path**: Uses `esxi_remote_cmds.go` to execute `vim-cmd`, `esxcli`, etc. over SSH
- **govmomi path**: Uses `govmomi_client.go` (connection/session management) and `govmomi_helpers.go` (shared utilities like VM lookup, power ops, task waiting)
- **Config** (`config.go`): Holds connection credentials and a cached `GovmomiClient`

### Resource File Organization

Each of the 5 resources (`guest`, `virtual_disk`, `resource_pool`, `vswitch`, `portgroup`) follows this pattern:

| File | Purpose |
|------|---------|
| `resource_<type>.go` | Terraform schema definition |
| `<type>-create.go` | Create operation |
| `<type>-read.go` | Read operation |
| `<type>_update.go` or `<type>-update.go` | Update operation |
| `<type>-delete.go` | Delete operation |
| `<type>-import.go` | Import state |
| `<type>_functions.go` | Core logic (contains both SSH and govmomi implementations) |

The single data source (`esxi_guest`) is in `data_source_esxi_guest.go`.

### Provider Entry Points

- `esxi_main.go` — binary entry point and `ConnectionStruct` definition
- `provider.go` — registers all resources/data sources and configures the provider (hostname, ports, credentials)

### Testing

Tests use `github.com/vmware/govmomi/simulator` (vcsim) to simulate an ESXi host — no real hardware needed. Tests use `simulator.ESX()` for standalone ESXi simulation. Test files follow the naming pattern `<type>_functions_test.go`.

### Guest VM Specifics

- Supports up to 10 network interfaces per VM
- SCSI disk slots validated in range `0:1` to `3:15` (ID 7 reserved)
- Cloud-init support via guestinfo metadata/userdata/vendordata with base64/gzip encoding
- VMX template handling for OVF source deployments
