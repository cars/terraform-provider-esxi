package esxi

import (
	"context"
	"fmt"
	"time"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// waitForTask waits for a task to complete with a timeout
func waitForTask(ctx context.Context, task *object.Task) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	return task.Wait(ctx)
}

// getVMByName finds a VM by its name
func getVMByName(ctx context.Context, finder *find.Finder, name string) (*object.VirtualMachine, error) {
	vm, err := finder.VirtualMachine(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' not found: %w", name, err)
	}
	return vm, nil
}

// getVMByID finds a VM by its managed object ID
func getVMByID(gc *GovmomiClient, id string) (*object.VirtualMachine, error) {
	moRef := types.ManagedObjectReference{
		Type:  "VirtualMachine",
		Value: id,
	}
	return object.NewVirtualMachine(gc.Client.Client, moRef), nil
}

// getPowerState returns the current power state of a VM
func getPowerState(ctx context.Context, vm *object.VirtualMachine) (types.VirtualMachinePowerState, error) {
	var mo mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"runtime.powerState"}, &mo)
	if err != nil {
		return "", fmt.Errorf("failed to get power state: %w", err)
	}
	return mo.Runtime.PowerState, nil
}

// getGuestIPAddress retrieves the IP address from VMware Tools
func getGuestIPAddress(ctx context.Context, vm *object.VirtualMachine) (string, error) {
	var mo mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"guest.ipAddress"}, &mo)
	if err != nil {
		return "", fmt.Errorf("failed to get guest IP: %w", err)
	}

	if mo.Guest == nil || mo.Guest.IpAddress == "" {
		return "", fmt.Errorf("guest IP address not available")
	}

	return mo.Guest.IpAddress, nil
}

// waitForGuestIPAddress waits for a VM to get an IP address with timeout
func waitForGuestIPAddress(ctx context.Context, vm *object.VirtualMachine, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var mo mo.VirtualMachine
	pc := property.DefaultCollector(vm.Client())

	err := property.Wait(ctx, pc, vm.Reference(), []string{"guest.ipAddress"}, func(pc []types.PropertyChange) bool {
		for _, c := range pc {
			if c.Val == nil {
				continue
			}
			if ip, ok := c.Val.(string); ok && ip != "" {
				mo.Guest = &types.GuestInfo{
					IpAddress: ip,
				}
				return true
			}
		}
		return false
	})

	if err != nil {
		return "", fmt.Errorf("timeout waiting for IP address: %w", err)
	}

	return mo.Guest.IpAddress, nil
}

// getDatastoreByName finds a datastore by name
func getDatastoreByName(ctx context.Context, finder *find.Finder, name string) (*object.Datastore, error) {
	ds, err := finder.Datastore(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("datastore '%s' not found: %w", name, err)
	}
	return ds, nil
}

// isDatastoreAccessible checks if a datastore is accessible
func isDatastoreAccessible(ctx context.Context, ds *object.Datastore) (bool, error) {
	var mo mo.Datastore
	err := ds.Properties(ctx, ds.Reference(), []string{"summary.accessible"}, &mo)
	if err != nil {
		return false, fmt.Errorf("failed to check datastore accessibility: %w", err)
	}
	return mo.Summary.Accessible, nil
}

// getHostSystem returns the default host system for standalone ESXi
func getHostSystem(ctx context.Context, finder *find.Finder) (*object.HostSystem, error) {
	host, err := finder.DefaultHostSystem(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find host system: %w", err)
	}
	return host, nil
}

// getResourcePool returns the root resource pool for the host
func getResourcePool(ctx context.Context, host *object.HostSystem) (*object.ResourcePool, error) {
	pool, err := host.ResourcePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource pool: %w", err)
	}
	return pool, nil
}

// getNetworkByName finds a network by name
func getNetworkByName(ctx context.Context, finder *find.Finder, name string) (object.NetworkReference, error) {
	network, err := finder.Network(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("network '%s' not found: %w", name, err)
	}
	return network, nil
}

// powerOnVM powers on a VM and waits for completion
func powerOnVM(ctx context.Context, vm *object.VirtualMachine) error {
	task, err := vm.PowerOn(ctx)
	if err != nil {
		return fmt.Errorf("failed to start power on task: %w", err)
	}

	err = waitForTask(ctx, task)
	if err != nil {
		return fmt.Errorf("power on task failed: %w", err)
	}

	return nil
}

// powerOffVM powers off a VM and waits for completion
func powerOffVM(ctx context.Context, vm *object.VirtualMachine) error {
	task, err := vm.PowerOff(ctx)
	if err != nil {
		return fmt.Errorf("failed to start power off task: %w", err)
	}

	err = waitForTask(ctx, task)
	if err != nil {
		return fmt.Errorf("power off task failed: %w", err)
	}

	return nil
}

// shutdownGuest initiates a graceful guest shutdown
func shutdownGuest(ctx context.Context, vm *object.VirtualMachine) error {
	err := vm.ShutdownGuest(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown guest: %w", err)
	}
	return nil
}

// destroyVM destroys a VM
func destroyVM(ctx context.Context, vm *object.VirtualMachine) error {
	task, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to start destroy task: %w", err)
	}

	err = waitForTask(ctx, task)
	if err != nil {
		return fmt.Errorf("destroy task failed: %w", err)
	}

	return nil
}

// reconfigureVM reconfigures a VM with the given spec
func reconfigureVM(ctx context.Context, vm *object.VirtualMachine, spec types.VirtualMachineConfigSpec) error {
	task, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to start reconfigure task: %w", err)
	}

	err = waitForTask(ctx, task)
	if err != nil {
		return fmt.Errorf("reconfigure task failed: %w", err)
	}

	return nil
}

// getHostNetworkSystem returns the network system for the host
func getHostNetworkSystem(ctx context.Context, host *object.HostSystem) (*object.HostNetworkSystem, error) {
	ns, err := host.ConfigManager().NetworkSystem(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get network system: %w", err)
	}
	return ns, nil
}
