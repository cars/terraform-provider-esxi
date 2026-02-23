# ESXi Data Sources Usage Guide

This guide demonstrates how to use the ESXi provider's data sources to discover and reference existing ESXi resources in your Terraform configurations.

## Available Data Sources

The ESXi provider now includes the following data sources:

- `esxi_host` - ESXi host system information
- `esxi_resource_pool` - Resource pool lookup
- `esxi_vswitch` - Virtual switch lookup  
- `esxi_portgroup` - Port group lookup
- `esxi_virtual_disk` - Virtual disk lookup
- `esxi_guest` - Guest VM lookup (existing)

## Basic Usage Examples

### ESXi Host Information

```hcl
# Get complete ESXi host information
data "esxi_host" "host" {}

output "host_info" {
  value = {
    hostname     = data.esxi_host.host.hostname
    version      = data.esxi_host.host.version
    cpu_cores    = data.esxi_host.host.cpu_cores
    memory_mb    = data.esxi_host.host.memory_size
    datastores   = data.esxi_host.host.datastores
  }
}
```

### Resource Pool Lookup

```hcl
# Look up a specific resource pool
data "esxi_resource_pool" "production" {
  resource_pool_name = "Production"
}

# Use the resource pool ID for new VMs
resource "esxi_guest" "new_vm" {
  guest_name         = "my-vm"
  resource_pool_name = data.esxi_resource_pool.production.resource_pool_name
  # ... other configuration
}
```

### Virtual Switch Discovery

```hcl
# Look up vSwitch0 configuration
data "esxi_vswitch" "main_switch" {
  name = "vSwitch0"
}

output "switch_info" {
  value = {
    name        = data.esxi_vswitch.main_switch.name
    ports       = data.esxi_vswitch.main_switch.ports
    mtu         = data.esxi_vswitch.main_switch.mtu
    uplinks     = data.esxi_vswitch.main_switch.uplink
  }
}
```

### Port Group Lookup

```hcl
# Find VM Network port group
data "esxi_portgroup" "vm_network" {
  name = "VM Network"
}

# Use for new VM network interfaces
resource "esxi_guest" "vm" {
  guest_name = "my-vm"
  network_interfaces {
    virtual_network = data.esxi_portgroup.vm_network.name
  }
  # ... other configuration
}
```

### Virtual Disk Discovery

```hcl
# Look up specific virtual disk
data "esxi_virtual_disk" "existing_disk" {
  virtual_disk_disk_store = "datastore1"
  virtual_disk_dir         = "my-vm"
  virtual_disk_name        = "disk1.vmdk"
}

# Auto-discover first .vmdk file in directory
data "esxi_virtual_disk" "auto_disk" {
  virtual_disk_disk_store = "datastore1"
  virtual_disk_dir         = "my-vm"
  # virtual_disk_name omitted for auto-discovery
}
```

## Advanced Patterns

### 1. Environment Discovery

Use data sources to automatically discover your ESXi environment:

```hcl
data "esxi_host" "host" {}
data "esxi_vswitch" "vswitch0" { name = "vSwitch0" }
data "esxi_portgroup" "vm_network" { name = "VM Network" }

resource "esxi_guest" "new_vm" {
  guest_name = "auto-discovered-vm"
  
  # Use discovered settings
  resource_pool_name = "Resources"
  network_interfaces {
    virtual_network = data.esxi_portgroup.vm_network.name
  }
  disk_store = data.esxi_host.host.datastores[0].name
  
  memsize  = 2048
  numvcpus = 2
}
```

### 2. Conditional Resource Creation

Create resources based on discovered host capabilities:

```hcl
data "esxi_host" "host" {}

# Only create high-memory VM if host has sufficient memory
resource "esxi_guest" "large_vm" {
  count = data.esxi_host.host.memory_size > 32768 ? 1 : 0
  
  guest_name = "large-vm"
  memsize    = 8192
  numvcpus   = 4
  # ... other configuration
}
```

### 3. Resource Templates

Use data sources to create templates for consistent resource creation:

```hcl
# Discover environment
data "esxi_host" "host" {}
data "esxi_vswitch" "vswitch0" { name = "vSwitch0" }
data "esxi_portgroup" "vm_network" { name = "VM Network" }

# Create multiple VMs with consistent settings
resource "esxi_guest" "web_servers" {
  count = 3
  guest_name = "web-server-${count.index + 1}"
  
  # Consistent configuration
  resource_pool_name = "Resources"
  network_interfaces {
    virtual_network = data.esxi_portgroup.vm_network.name
  }
  disk_store = data.esxi_host.host.datastores[0].name
  
  memsize  = 2048
  numvcpus = 2
}
```

### 4. Network Configuration Discovery

Discover complete network configuration:

```hcl
data "esxi_vswitch" "all_switches" {
  # You can create multiple data source instances
  # for different switches if needed
}

data "esxi_portgroup" "all_portgroups" {
  # Create instances for each portgroup you want to discover
}

output "network_topology" {
  value = {
    vswitches = [
      {
        name    = data.esxi_vswitch.vswitch0.name
        ports   = data.esxi_vswitch.vswitch0.ports
        mtu     = data.esxi_vswitch.vswitch0.mtu
        uplinks = data.esxi_vswitch.vswitch0.uplink
      }
    ]
    portgroups = [
      {
        name    = data.esxi_portgroup.vm_network.name
        vswitch = data.esxi_portgroup.vm_network.vswitch
        vlan    = data.esxi_portgroup.vm_network.vlan
      }
    ]
  }
}
```

## Data Source Reference

### esxi_host

Returns complete ESXi host system information.

**Read-only attributes:**
- `hostname` - ESXi host hostname or IP
- `version` - ESXi version and build
- `product_name` - Product name (e.g., "VMware ESXi")
- `uuid` - Host system UUID
- `manufacturer` - Hardware manufacturer
- `model` - Hardware model
- `serial_number` - Hardware serial number
- `cpu_model` - CPU model description
- `cpu_packages` - Number of CPU packages
- `cpu_cores` - Number of CPU cores
- `cpu_threads` - Number of CPU threads
- `cpu_mhz` - CPU speed in MHz
- `memory_size` - Total memory in MB
- `datastores` - List of datastores

### esxi_resource_pool

Look up resource pool by name.

**Required:**
- `resource_pool_name` - Name of the resource pool

**Read-only attributes:**
- `cpu_min` - CPU minimum in MHz
- `cpu_min_expandable` - Can borrow CPU from parent
- `cpu_max` - CPU maximum in MHz
- `cpu_shares` - CPU shares level
- `mem_min` - Memory minimum in MB
- `mem_min_expandable` - Can borrow memory from parent
- `mem_max` - Memory maximum in MB
- `mem_shares` - Memory shares level

### esxi_vswitch

Look up virtual switch by name.

**Required:**
- `name` - Virtual switch name

**Read-only attributes:**
- `ports` - Number of ports
- `mtu` - MTU size
- `link_discovery_mode` - Link discovery mode
- `promiscuous_mode` - Promiscuous mode setting
- `mac_changes` - MAC changes setting
- `forged_transmits` - Forged transmits setting
- `uplink` - List of uplinks

### esxi_portgroup

Look up port group by name.

**Required:**
- `name` - Port group name

**Read-only attributes:**
- `vswitch` - Parent virtual switch
- `vlan` - VLAN ID
- `promiscuous_mode` - Promiscuous mode setting
- `mac_changes` - MAC changes setting
- `forged_transmits` - Forged transmits setting

### esxi_virtual_disk

Look up virtual disk by location.

**Required:**
- `virtual_disk_disk_store` - Datastore name
- `virtual_disk_dir` - Directory path

**Optional:**
- `virtual_disk_name` - Specific disk name (auto-discovery if omitted)

**Read-only attributes:**
- `virtual_disk_size` - Disk size in GB
- `virtual_disk_type` - Disk type (thin, zeroedthick, eagerzeroedthick)

## Best Practices

1. **Use descriptive variable names** for data source instances
2. **Combine multiple data sources** to discover complete environments
3. **Use data sources for consistency** when creating multiple similar resources
4. **Implement conditional logic** based on discovered host capabilities
5. **Document your discoveries** with outputs for debugging and planning

## Migration Tips

If you're migrating from existing configurations:

1. Replace hardcoded values with data source lookups
2. Use data sources to validate resource existence before creation
3. Implement gradual migration by starting with read-only data sources
4. Use outputs to verify discovered values match expectations

These data sources make your Terraform configurations more dynamic and adaptable to different ESXi environments!