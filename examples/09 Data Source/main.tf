provider "esxi" {
  esxi_hostname = var.esxi_hostname
  esxi_hostport = var.esxi_hostport
  esxi_username = var.esxi_username
  esxi_password = var.esxi_password
}

# Example 1: Look up an existing VM by name
data "esxi_guest" "existing_vm" {
  #guest_name = "my-existing-vm"
  guest_name = "lab-router"
}

# Example 2: Look up a VM by its VM ID
# Uncomment to use:
# data "esxi_guest" "by_id" {
#   vmid = "42"
# }

# Display VM information
output "vm_details" {
  description = "Details of the existing VM"
  value = {
    id           = data.esxi_guest.existing_vm.id
    name         = data.esxi_guest.existing_vm.guest_name
    memory_mb    = data.esxi_guest.existing_vm.memsize
    cpus         = data.esxi_guest.existing_vm.numvcpus
    power_state  = data.esxi_guest.existing_vm.power
    ip_address   = data.esxi_guest.existing_vm.ip_address
    datastore    = data.esxi_guest.existing_vm.disk_store
    disk_size_gb = data.esxi_guest.existing_vm.boot_disk_size
    hw_version   = data.esxi_guest.existing_vm.virthwver
    guest_os     = data.esxi_guest.existing_vm.guestos
  }
}

# Display network interface details
output "vm_network_interfaces" {
  description = "Network interfaces of the existing VM"
  value       = data.esxi_guest.existing_vm.network_interfaces
}

# Display virtual disks
output "vm_virtual_disks" {
  description = "Virtual disks attached to the VM"
  value       = data.esxi_guest.existing_vm.virtual_disks
}

# Example 3: Use data source values to create a similar VM
resource "esxi_guest" "cloned_vm" {
  guest_name = "cloned-vm"

  # Reuse configuration from existing VM
  disk_store = data.esxi_guest.existing_vm.disk_store
  memsize    = data.esxi_guest.existing_vm.memsize
  numvcpus   = data.esxi_guest.existing_vm.numvcpus
  virthwver  = data.esxi_guest.existing_vm.virthwver
  guestos    = data.esxi_guest.existing_vm.guestos

  # Optionally clone the network configuration
  dynamic "network_interfaces" {
    for_each = data.esxi_guest.existing_vm.network_interfaces
    content {
      virtual_network = network_interfaces.value.virtual_network
      nic_type        = network_interfaces.value.nic_type
    }
  }

  # Set power state
  power = "on"
}
