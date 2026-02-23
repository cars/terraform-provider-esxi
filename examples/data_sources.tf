# ESXi Data Sources Examples
# 
# This file demonstrates how to use all the ESXi data sources for
# discovering and referencing existing ESXi resources.

# =============================================================================
# ESXi Host Information
# =============================================================================
# Get complete information about the ESXi host including hardware specs,
# version, memory, CPU, and available datastores
data "esxi_host" "host_info" {
}

output "esxi_host_details" {
  description = "ESXi host system information"
  value = {
    hostname       = data.esxi_host.host_info.hostname
    version        = data.esxi_host.host_info.version
    product_name   = data.esxi_host.host_info.product_name
    uuid           = data.esxi_host.host_info.uuid
    manufacturer   = data.esxi_host.host_info.manufacturer
    model          = data.esxi_host.host_info.model
    cpu_model      = data.esxi_host.host_info.cpu_model
    cpu_cores      = data.esxi_host.host_info.cpu_cores
    cpu_threads    = data.esxi_host.host_info.cpu_threads
    memory_size_mb = data.esxi_host.host_info.memory_size
    datastores     = data.esxi_host.host_info.datastores
  }
}

# =============================================================================
# Resource Pool Examples
# =============================================================================

# Example 1: Look up a specific resource pool by name
data "esxi_resource_pool" "production_pool" {
  resource_pool_name = "Production"
}

output "production_pool_info" {
  description = "Production resource pool allocation settings"
  value = {
    name        = data.esxi_resource_pool.production_pool.resource_pool_name
    cpu_min_mhz = data.esxi_resource_pool.production_pool.cpu_min
    cpu_max_mhz = data.esxi_resource_pool.production_pool.cpu_max
    cpu_shares  = data.esxi_resource_pool.production_pool.cpu_shares
    mem_min_mb  = data.esxi_resource_pool.production_pool.mem_min
    mem_max_mb  = data.esxi_resource_pool.production_pool.mem_max
    mem_shares  = data.esxi_resource_pool.production_pool.mem_shares
  }
}

# Example 2: Look up the root resource pool
data "esxi_resource_pool" "root_pool" {
  resource_pool_name = "Resources"
}

# =============================================================================
# Virtual Switch Examples
# =============================================================================

# Example 1: Look up the default vSwitch0
data "esxi_vswitch" "vswitch0" {
  name = "vSwitch0"
}

output "vswitch0_details" {
  description = "vSwitch0 configuration details"
  value = {
    name                = data.esxi_vswitch.vswitch0.name
    ports               = data.esxi_vswitch.vswitch0.ports
    mtu                 = data.esxi_vswitch.vswitch0.mtu
    link_discovery_mode = data.esxi_vswitch.vswitch0.link_discovery_mode
    promiscuous_mode    = data.esxi_vswitch.vswitch0.promiscuous_mode
    mac_changes         = data.esxi_vswitch.vswitch0.mac_changes
    forged_transmits    = data.esxi_vswitch.vswitch0.forged_transmits
    uplinks             = data.esxi_vswitch.vswitch0.uplink
  }
}

# Example 2: Look up a custom vSwitch
data "esxi_vswitch" "storage_network" {
  name = "StorageNetwork"
}

# =============================================================================
# Port Group Examples
# =============================================================================

# Example 1: Look up the VM Network port group
data "esxi_portgroup" "vm_network" {
  name = "VM Network"
}

output "vm_network_details" {
  description = "VM Network port group configuration"
  value = {
    name             = data.esxi_portgroup.vm_network.name
    vswitch          = data.esxi_portgroup.vm_network.vswitch
    vlan             = data.esxi_portgroup.vm_network.vlan
    promiscuous_mode = data.esxi_portgroup.vm_network.promiscuous_mode
    mac_changes      = data.esxi_portgroup.vm_network.mac_changes
    forged_transmits = data.esxi_portgroup.vm_network.forged_transmits
  }
}

# Example 2: Look up a management port group
data "esxi_portgroup" "management" {
  name = "Management Network"
}

# =============================================================================
# Virtual Disk Examples
# =============================================================================

# Example 1: Look up a specific virtual disk by name
data "esxi_virtual_disk" "web_server_disk" {
  virtual_disk_disk_store = "datastore1"
  virtual_disk_dir        = "web-server"
  virtual_disk_name       = "web-server.vmdk"
}

output "web_server_disk_info" {
  description = "Web server virtual disk details"
  value = {
    disk_store = data.esxi_virtual_disk.web_server_disk.virtual_disk_disk_store
    directory  = data.esxi_virtual_disk.web_server_disk.virtual_disk_dir
    name       = data.esxi_virtual_disk.web_server_disk.virtual_disk_name
    size_gb    = data.esxi_virtual_disk.web_server_disk.virtual_disk_size
    type       = data.esxi_virtual_disk.web_server_disk.virtual_disk_type
  }
}

# Example 2: Auto-discover virtual disk in directory (finds first .vmdk file)
data "esxi_virtual_disk" "database_disk" {
  virtual_disk_disk_store = "datastore1"
  virtual_disk_dir        = "database-server"
  # virtual_disk_name is omitted - will auto-discover first .vmdk file
}

# =============================================================================
# Guest VM Examples (using existing data source)
# =============================================================================

# Example 1: Look up guest VM by name
data "esxi_guest" "web_server" {
  guest_name = "web-server-01"
}

output "web_server_info" {
  description = "Web server VM information"
  value = {
    name              = data.esxi_guest.web_server.guest_name
    power_state       = data.esxi_guest.web_server.power
    memory_size_mb    = data.esxi_guest.web_server.memsize
    cpu_count         = data.esxi_guest.web_server.numvcpus
    guest_os          = data.esxi_guest.web_server.guestos
    boot_disk_type    = data.esxi_guest.web_server.boot_disk_type
    boot_disk_size_gb = data.esxi_guest.web_server.boot_disk_size
    resource_pool     = data.esxi_guest.web_server.resource_pool_name
    ip_address        = data.esxi_guest.web_server.ip_address
  }
}

# Example 2: Look up guest VM by ID
data "esxi_guest" "database_server" {
  vmid = "562d3a2b-5f1b-4c3a-9b8d-7e6f5a4b3c2d"
}

# =============================================================================
# Advanced Usage Examples
# =============================================================================

# Example 1: Use data sources to create new resources with discovered settings
resource "esxi_guest" "new_vm" {
  guest_name         = "new-vm-from-template"
  resource_pool_name = data.esxi_resource_pool.production_pool.resource_pool_name

  # Use discovered network information
  network_interfaces {
    virtual_network = data.esxi_portgroup.vm_network.name
  }

  # Use discovered datastore information
  disk_store = data.esxi_host.host_info.datastores[0].name

  memsize  = 2048
  numvcpus = 2
}

# Example 2: Create a new port group on an existing vSwitch
resource "esxi_portgroup" "new_portgroup" {
  name    = "ApplicationNetwork"
  vswitch = data.esxi_vswitch.vswitch0.name
  vlan    = 100

  # Inherit security settings from existing port group
  promiscuous_mode = data.esxi_portgroup.vm_network.promiscuous_mode
  mac_changes      = data.esxi_portgroup.vm_network.mac_changes
  forged_transmits = data.esxi_portgroup.vm_network.forged_transmits
}

# Example 3: Create virtual disk in same directory as existing disk
resource "esxi_virtual_disk" "additional_disk" {
  virtual_disk_disk_store = data.esxi_virtual_disk.web_server_disk.virtual_disk_disk_store
  virtual_disk_dir        = data.esxi_virtual_disk.web_server_disk.virtual_disk_dir
  virtual_disk_name       = "additional-disk.vmdk"
  virtual_disk_size       = 50
  virtual_disk_type       = "thin"
}

# =============================================================================
# Conditional Resource Creation
# =============================================================================

# Example: Only create resources if certain conditions are met
resource "esxi_guest" "conditional_vm" {
  count = data.esxi_host.host_info.memory_size > 32768 ? 1 : 0

  guest_name = "high-memory-vm"
  memsize    = 8192
  numvcpus   = 4

  resource_pool_name = data.esxi_resource_pool.root_pool.resource_pool_name
  network_interfaces {
    virtual_network = data.esxi_portgroup.vm_network.name
  }
  disk_store = data.esxi_host.host_info.datastores[0].name
}

# =============================================================================
# Output All Discovered Information
# =============================================================================

output "discovered_esxi_environment" {
  description = "Complete discovered ESXi environment"
  value = {
    host = {
      hostname     = data.esxi_host.host_info.hostname
      version      = data.esxi_host.host_info.version
      manufacturer = data.esxi_host.host_info.manufacturer
      model        = data.esxi_host.host_info.model
      cpu_cores    = data.esxi_host.host_info.cpu_cores
      memory_mb    = data.esxi_host.host_info.memory_size
    }

    networking = {
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

    storage = {
      datastores = data.esxi_host.host_info.datastores
      virtual_disks = [
        {
          name    = data.esxi_virtual_disk.web_server_disk.virtual_disk_name
          size_gb = data.esxi_virtual_disk.web_server_disk.virtual_disk_size
          type    = data.esxi_virtual_disk.web_server_disk.virtual_disk_type
        }
      ]
    }

    compute = {
      resource_pools = [
        {
          name       = data.esxi_resource_pool.production_pool.resource_pool_name
          cpu_shares = data.esxi_resource_pool.production_pool.cpu_shares
          mem_shares = data.esxi_resource_pool.production_pool.mem_shares
        }
      ]
    }
  }
}