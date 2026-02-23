package esxi

import (
	"fmt"
	"log"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

//  Python is better... :-)
func inArrayOfStrings(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// ============================================================================
// Network Switch Operations
// ============================================================================

// vswitchCreate creates a vswitch using govmomi
func vswitchCreate(c *Config, name string, ports int) error {
	log.Printf("[vswitchCreate] Creating vswitch %s\n", name)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return fmt.Errorf("failed to get govmomi client: %w", err)
	}

	host, err := getHostSystem(gc.Context(), gc.Finder)
	if err != nil {
		return fmt.Errorf("failed to get host system: %w", err)
	}

	ns, err := getHostNetworkSystem(gc.Context(), host)
	if err != nil {
		return fmt.Errorf("failed to get network system: %w", err)
	}

	// Create vswitch spec
	spec := types.HostVirtualSwitchSpec{
		NumPorts: int32(ports),
	}

	err = ns.AddVirtualSwitch(gc.Context(), name, &spec)
	if err != nil {
		return fmt.Errorf("failed to create vswitch: %w", err)
	}

	log.Printf("[vswitchCreate] Successfully created vswitch %s\n", name)
	return nil
}

// vswitchDelete deletes a vswitch using govmomi
func vswitchDelete(c *Config, name string) error {
	log.Printf("[vswitchDelete] Deleting vswitch %s\n", name)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return fmt.Errorf("failed to get govmomi client: %w", err)
	}

	host, err := getHostSystem(gc.Context(), gc.Finder)
	if err != nil {
		return fmt.Errorf("failed to get host system: %w", err)
	}

	ns, err := getHostNetworkSystem(gc.Context(), host)
	if err != nil {
		return fmt.Errorf("failed to get network system: %w", err)
	}

	err = ns.RemoveVirtualSwitch(gc.Context(), name)
	if err != nil {
		return fmt.Errorf("failed to delete vswitch: %w", err)
	}

	log.Printf("[vswitchDelete] Successfully deleted vswitch %s\n", name)
	return nil
}

// vswitchRead reads vswitch configuration using govmomi
func vswitchRead(c *Config, name string) (int, int, []string, string, bool, bool, bool, error) {
	log.Printf("[vswitchRead] Reading vswitch %s\n", name)

	var ports, mtu int
	var uplinks []string
	var link_discovery_mode string
	var promiscuous_mode, mac_changes, forged_transmits bool

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf("failed to get govmomi client: %w", err)
	}

	host, err := getHostSystem(gc.Context(), gc.Finder)
	if err != nil {
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf("failed to get host system: %w", err)
	}

	ns, err := getHostNetworkSystem(gc.Context(), host)
	if err != nil {
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf("failed to get network system: %w", err)
	}

	// Get network configuration
	var hostNetworkSystem mo.HostNetworkSystem
	err = ns.Properties(gc.Context(), ns.Reference(), []string{"networkInfo"}, &hostNetworkSystem)
	if err != nil {
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf("failed to get network info: %w", err)
	}

	// Find the vswitch
	var vswitch *types.HostVirtualSwitch
	if hostNetworkSystem.NetworkInfo != nil {
		for i := range hostNetworkSystem.NetworkInfo.Vswitch {
			if hostNetworkSystem.NetworkInfo.Vswitch[i].Name == name {
				vswitch = &hostNetworkSystem.NetworkInfo.Vswitch[i]
				break
			}
		}
	}

	if vswitch == nil {
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf("vswitch %s not found", name)
	}

	// Get basic properties
	ports = int(vswitch.Spec.NumPorts)
	mtu = int(vswitch.Mtu)

	// Get uplinks
	if vswitch.Pnic != nil {
		uplinks = vswitch.Pnic
	}

	// Get link discovery mode (CDP/LLDP)
	if vswitch.Spec.Bridge != nil {
		if linkDiscoveryProtocolConfig, ok := vswitch.Spec.Bridge.(*types.HostVirtualSwitchBondBridge); ok {
			if linkDiscoveryProtocolConfig.LinkDiscoveryProtocolConfig != nil {
				protocol := linkDiscoveryProtocolConfig.LinkDiscoveryProtocolConfig.Protocol
				operation := linkDiscoveryProtocolConfig.LinkDiscoveryProtocolConfig.Operation

				if protocol == "cdp" {
					if operation == "listen" {
						link_discovery_mode = "listen"
					} else if operation == "advertise" {
						link_discovery_mode = "advertise"
					} else if operation == "both" {
						link_discovery_mode = "both"
					} else {
						link_discovery_mode = "down"
					}
				} else {
					link_discovery_mode = "down"
				}
			} else {
				link_discovery_mode = "listen"
			}
		} else {
			link_discovery_mode = "listen"
		}
	} else {
		link_discovery_mode = "listen"
	}

	// Get security policy
	if vswitch.Spec.Policy != nil && vswitch.Spec.Policy.Security != nil {
		if vswitch.Spec.Policy.Security.AllowPromiscuous != nil {
			promiscuous_mode = *vswitch.Spec.Policy.Security.AllowPromiscuous
		}
		if vswitch.Spec.Policy.Security.MacChanges != nil {
			mac_changes = *vswitch.Spec.Policy.Security.MacChanges
		}
		if vswitch.Spec.Policy.Security.ForgedTransmits != nil {
			forged_transmits = *vswitch.Spec.Policy.Security.ForgedTransmits
		}
	}

	log.Printf("[vswitchRead] Successfully read vswitch %s\n", name)
	return ports, mtu, uplinks, link_discovery_mode, promiscuous_mode, mac_changes, forged_transmits, nil
}

// vswitchUpdate updates vswitch configuration using govmomi
func vswitchUpdate(c *Config, name string, ports int, mtu int, uplinks []string,
	link_discovery_mode string, promiscuous_mode bool, mac_changes bool, forged_transmits bool) error {
	log.Printf("[vswitchUpdate] Updating vswitch %s\n", name)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return fmt.Errorf("failed to get govmomi client: %w", err)
	}

	host, err := getHostSystem(gc.Context(), gc.Finder)
	if err != nil {
		return fmt.Errorf("failed to get host system: %w", err)
	}

	ns, err := getHostNetworkSystem(gc.Context(), host)
	if err != nil {
		return fmt.Errorf("failed to get network system: %w", err)
	}

	// Build link discovery protocol config
	var linkDiscoveryProtocolConfig *types.LinkDiscoveryProtocolConfig
	if link_discovery_mode != "down" {
		linkDiscoveryProtocolConfig = &types.LinkDiscoveryProtocolConfig{
			Protocol:  "cdp",
			Operation: link_discovery_mode,
		}
	}

	// Build bridge spec with uplinks
	bridge := &types.HostVirtualSwitchBondBridge{
		HostVirtualSwitchBridge: types.HostVirtualSwitchBridge{},
		NicDevice:               uplinks,
		LinkDiscoveryProtocolConfig: linkDiscoveryProtocolConfig,
	}

	// Build security policy
	securityPolicy := &types.HostNetworkSecurityPolicy{
		AllowPromiscuous: &promiscuous_mode,
		MacChanges:       &mac_changes,
		ForgedTransmits:  &forged_transmits,
	}

	// Build vswitch spec
	spec := types.HostVirtualSwitchSpec{
		NumPorts: int32(ports),
		Mtu:      int32(mtu),
		Bridge:   bridge,
		Policy: &types.HostNetworkPolicy{
			Security: securityPolicy,
		},
	}

	// Update the vswitch (this updates everything including uplinks)
	err = ns.UpdateVirtualSwitch(gc.Context(), name, spec)
	if err != nil {
		return fmt.Errorf("failed to update vswitch: %w", err)
	}

	log.Printf("[vswitchUpdate] Successfully updated vswitch %s\n", name)
	return nil
}
