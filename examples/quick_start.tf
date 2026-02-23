# Quick Start Examples

These are minimal examples to get you started with the new ESXi data sources.

## Example 1: Basic Host Discovery

```hcl
# Configure the ESXi provider
provider "esxi" {
  esxi_hostname = "your-esxi-host.local"
  esxi_username = "root"
  esxi_password = "your-password"
}

# Discover host information
data "esxi_host" "host" {}

# Output basic host info
output "host_summary" {
  value = {
    hostname  = data.esxi_host.host.hostname
    version   = data.esxi_host.host.version
    cpu_cores = data.esxi_host.host.cpu_cores
    memory_gb = data.esxi_host.host.memory_size / 1024
  }
}
```

## Example 2: Network Discovery

```hcl
provider "esxi" {
  esxi_hostname = "your-esxi-host.local"
  esxi_username = "root"
  esxi_password = "your-password"
}

# Discover network configuration
data "esxi_vswitch" "vswitch0" {
  name = "vSwitch0"
}

data "esxi_portgroup" "vm_network" {
  name = "VM Network"
}

output "network_info" {
  value = {
    switch_name = data.esxi_vswitch.vswitch0.name
    switch_ports = data.esxi_vswitch.vswitch0.ports
    portgroup_name = data.esxi_portgroup.vm_network.name
    portgroup_vlan = data.esxi_portgroup.vm_network.vlan
  }
}
```

## Example 3: Create VM Using Discovered Settings

```hcl
provider "esxi" {
  esxi_hostname = "your-esxi-host.local"
  esxi_username = "root"
  esxi_password = "your-password"
}

# Discover existing resources
data "esxi_host" "host" {}
data "esxi_portgroup" "network" {
  name = "VM Network"
}

# Create new VM using discovered settings
resource "esxi_guest" "new_vm" {
  guest_name = "my-new-vm"
  
  # Use discovered network
  network_interfaces {
    virtual_network = data.esxi_portgroup.network.name
  }
  
  # Use discovered datastore
  disk_store = data.esxi_host.host.datastores[0].name
  
  memsize  = 2048
  numvcpus = 2
}

output "vm_info" {
  value = {
    name = resource.esxi_guest.new_vm.guest_name
    power = resource.esxi_guest.new_vm.power
    network = data.esxi_portgroup.network.name
    datastore = data.esxi_host.host.datastores[0].name
  }
}
```

## Example 4: Virtual Disk Discovery

```hcl
provider "esxi" {
  esxi_hostname = "your-esxi-host.local"
  esxi_username = "root"
  esxi_password = "your-password"
}

# Discover existing virtual disk
data "esxi_virtual_disk" "existing_disk" {
  virtual_disk_disk_store = "datastore1"
  virtual_disk_dir         = "existing-vm"
  # virtual_disk_name omitted for auto-discovery
}

output "disk_info" {
  value = {
    name    = data.esxi_virtual_disk.existing_disk.virtual_disk_name
    size_gb = data.esxi_virtual_disk.existing_disk.virtual_disk_size
    type    = data.esxi_virtual_disk.existing_disk.virtual_disk_type
  }
}
```

## Example 5: Resource Pool Discovery

```hcl
provider "esxi" {
  esxi_hostname = "your-esxi-host.local"
  esxi_username = "root"
  esxi_password = "your-password"
}

# Discover resource pool
data "esxi_resource_pool" "production" {
  resource_pool_name = "Production"
}

output "resource_pool_info" {
  value = {
    name        = data.esxi_resource_pool.production.resource_pool_name
    cpu_shares  = data.esxi_resource_pool.production.cpu_shares
    mem_shares  = data.esxi_resource_pool.production.mem_shares
    cpu_max_mhz = data.esxi_resource_pool.production.cpu_max
    mem_max_mb  = data.esxi_resource_pool.production.mem_max
  }
}
```

## Example 6: Complete Environment Scan

```hcl
provider "esxi" {
  esxi_hostname = "your-esxi-host.local"
  esxi_username = "root"
  esxi_password = "your-password"
}

# Discover everything
data "esxi_host" "host" {}
data "esxi_vswitch" "vswitch0" { name = "vSwitch0" }
data "esxi_portgroup" "vm_network" { name = "VM Network" }
data "esxi_resource_pool" "resources" { resource_pool_name = "Resources" }

output "environment_summary" {
  value = {
    host = {
      hostname  = data.esxi_host.host.hostname
      version   = data.esxi_host.host.version
      cpu_cores = data.esxi_host.host.cpu_cores
      memory_gb = data.esxi_host.host.memory_size / 1024
    }
    network = {
      vswitch_name = data.esxi_vswitch.vswitch0.name
      vswitch_ports = data.esxi_vswitch.vswitch0.ports
      portgroup_name = data.esxi_portgroup.vm_network.name
      portgroup_vlan = data.esxi_portgroup.vm_network.vlan
    }
    compute = {
      resource_pool = data.esxi_resource_pool.resources.resource_pool_name
      cpu_shares = data.esxi_resource_pool.resources.cpu_shares
      mem_shares = data.esxi_resource_pool.resources.mem_shares
    }
  }
}
```

## Running These Examples

1. Save any example to a file (e.g., `main.tf`)
2. Update the provider configuration with your ESXi host details
3. Run: `terraform init`
4. Run: `terraform plan`
5. Run: `terraform apply`

These examples demonstrate the basic patterns for using ESXi data sources in your Terraform configurations!