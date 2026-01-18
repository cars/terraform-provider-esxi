# ESXi Guest Data Source Example

This example demonstrates how to use the `esxi_guest` data source to query information about existing VMs on an ESXi host.

## What This Example Does

1. **Look up an existing VM by name** - Query VM configuration using its guest name
2. **Display VM details** - Output memory, CPUs, power state, IP address, and more
3. **Show network interfaces** - List all NICs with their network names and MAC addresses
4. **Show virtual disks** - Display attached virtual disks and their SCSI slots
5. **Create a similar VM** - Use data source values to clone the configuration

## Prerequisites

- An existing VM on your ESXi host (update `guest_name` in main.tf)
- ESXi host accessible via SSH
- Valid ESXi credentials

## Usage

### 1. Update the guest name

Edit `main.tf` and replace `"my-existing-vm"` with the name of an actual VM on your ESXi host:

```hcl
data "esxi_guest" "existing_vm" {
  guest_name = "your-actual-vm-name"
}
```

### 2. Set provider credentials

Either set environment variables:

```bash
export TF_VAR_esxi_password="your-esxi-password"
```

Or create a `terraform.tfvars` file:

```hcl
esxi_hostname = "192.168.1.10"
esxi_username = "root"
esxi_password = "your-password"
```

### 3. Initialize and apply

```bash
terraform init
terraform plan
terraform apply
```

### 4. View outputs

```bash
terraform output vm_details
terraform output vm_network_interfaces
terraform output vm_virtual_disks
```

## Looking Up by VM ID

If you know the ESXi VM ID, you can use that instead:

```hcl
data "esxi_guest" "by_id" {
  vmid = "42"
}
```

To find a VM's ID, SSH to your ESXi host and run:

```bash
vim-cmd vmsvc/getallvms
```

## Available Data Source Attributes

The data source provides the following information:

- **Basic Info**: `guest_name`, `disk_store`, `resource_pool_name`
- **Hardware**: `memsize`, `numvcpus`, `virthwver`, `guestos`, `boot_firmware`
- **Storage**: `boot_disk_size`, `boot_disk_type`, `virtual_disks`
- **Network**: `network_interfaces`, `ip_address`
- **State**: `power`
- **Metadata**: `notes`, `guestinfo`

## Notes

- The VM must already exist on the ESXi host
- IP address requires VMware Tools to be installed and running
- You can use either `guest_name` or `vmid`, but not both
- Data sources are read-only; they query existing infrastructure
