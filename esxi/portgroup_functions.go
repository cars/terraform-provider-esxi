package esxi

import (
	"fmt"
	"log"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type portgroupSecurityPolicy struct {
	AllowForgedTransmits  bool `csv:"AllowForgedTransmits"`
	AllowMACAddressChange bool `csv:"AllowMACAddressChange"`
	AllowPromiscuous      bool `csv:"AllowPromiscuous"`
}

// ============================================================================
// Port Group Operations
// ============================================================================

// portgroupCreate creates a port group using govmomi
func portgroupCreate(c *Config, name string, vswitch string) error {
	log.Printf("[portgroupCreate] Creating portgroup %s on vswitch %s\n", name, vswitch)

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

	// Create port group spec
	spec := types.HostPortGroupSpec{
		Name:        name,
		VswitchName: vswitch,
		VlanId:      0, // default VLAN
		Policy:      types.HostNetworkPolicy{},
	}

	err = ns.AddPortGroup(gc.Context(), spec)
	if err != nil {
		return fmt.Errorf("failed to create portgroup: %w", err)
	}

	log.Printf("[portgroupCreate] Successfully created portgroup %s\n", name)
	return nil
}

// portgroupDelete deletes a port group using govmomi
func portgroupDelete(c *Config, name string) error {
	log.Printf("[portgroupDelete] Deleting portgroup %s\n", name)

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

	err = ns.RemovePortGroup(gc.Context(), name)
	if err != nil {
		return fmt.Errorf("failed to delete portgroup: %w", err)
	}

	log.Printf("[portgroupDelete] Successfully deleted portgroup %s\n", name)
	return nil
}

// portgroupRead reads port group configuration using govmomi
func portgroupRead(c *Config, name string) (string, int, error) {
	log.Printf("[portgroupRead] Reading portgroup %s\n", name)

	var vswitch string
	var vlan int

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get govmomi client: %w", err)
	}

	host, err := getHostSystem(gc.Context(), gc.Finder)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get host system: %w", err)
	}

	ns, err := getHostNetworkSystem(gc.Context(), host)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get network system: %w", err)
	}

	// Get network configuration
	var hostNetworkSystem mo.HostNetworkSystem
	err = ns.Properties(gc.Context(), ns.Reference(), []string{"networkInfo"}, &hostNetworkSystem)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get network info: %w", err)
	}

	// Find the portgroup
	var portgroup *types.HostPortGroup
	if hostNetworkSystem.NetworkInfo != nil {
		for i := range hostNetworkSystem.NetworkInfo.Portgroup {
			if hostNetworkSystem.NetworkInfo.Portgroup[i].Spec.Name == name {
				portgroup = &hostNetworkSystem.NetworkInfo.Portgroup[i]
				break
			}
		}
	}

	if portgroup == nil {
		return "", 0, fmt.Errorf("portgroup %s not found", name)
	}

	vswitch = portgroup.Spec.VswitchName
	vlan = int(portgroup.Spec.VlanId)

	log.Printf("[portgroupRead] Successfully read portgroup %s\n", name)
	return vswitch, vlan, nil
}

// portgroupSecurityPolicyRead reads port group security policy using govmomi
func portgroupSecurityPolicyRead(c *Config, name string) (*portgroupSecurityPolicy, error) {
	log.Printf("[portgroupSecurityPolicyRead] Reading security policy for portgroup %s\n", name)

	gc, err := c.GetGovmomiClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get govmomi client: %w", err)
	}

	host, err := getHostSystem(gc.Context(), gc.Finder)
	if err != nil {
		return nil, fmt.Errorf("failed to get host system: %w", err)
	}

	ns, err := getHostNetworkSystem(gc.Context(), host)
	if err != nil {
		return nil, fmt.Errorf("failed to get network system: %w", err)
	}

	// Get network configuration
	var hostNetworkSystem mo.HostNetworkSystem
	err = ns.Properties(gc.Context(), ns.Reference(), []string{"networkInfo"}, &hostNetworkSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get network info: %w", err)
	}

	// Find the portgroup
	var portgroup *types.HostPortGroup
	if hostNetworkSystem.NetworkInfo != nil {
		for i := range hostNetworkSystem.NetworkInfo.Portgroup {
			if hostNetworkSystem.NetworkInfo.Portgroup[i].Spec.Name == name {
				portgroup = &hostNetworkSystem.NetworkInfo.Portgroup[i]
				break
			}
		}
	}

	if portgroup == nil {
		return nil, fmt.Errorf("portgroup %s not found", name)
	}

	// Extract security policy
	policy := &portgroupSecurityPolicy{}
	if portgroup.Spec.Policy.Security != nil {
		if portgroup.Spec.Policy.Security.AllowPromiscuous != nil {
			policy.AllowPromiscuous = *portgroup.Spec.Policy.Security.AllowPromiscuous
		}
		if portgroup.Spec.Policy.Security.MacChanges != nil {
			policy.AllowMACAddressChange = *portgroup.Spec.Policy.Security.MacChanges
		}
		if portgroup.Spec.Policy.Security.ForgedTransmits != nil {
			policy.AllowForgedTransmits = *portgroup.Spec.Policy.Security.ForgedTransmits
		}
	}

	log.Printf("[portgroupSecurityPolicyRead] Successfully read security policy\n")
	return policy, nil
}

// portgroupUpdate updates port group configuration using govmomi
func portgroupUpdate(c *Config, name string, vlan int, promiscuous_mode, forged_transmits, mac_changes string) error {
	log.Printf("[portgroupUpdate] Updating portgroup %s\n", name)

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

	// Get current portgroup info to build the complete spec
	vswitch, _, err := portgroupRead(c, name)
	if err != nil {
		return fmt.Errorf("failed to read portgroup: %w", err)
	}

	// Build security policy
	var securityPolicy *types.HostNetworkSecurityPolicy
	if promiscuous_mode != "" || forged_transmits != "" || mac_changes != "" {
		securityPolicy = &types.HostNetworkSecurityPolicy{}

		if promiscuous_mode != "" {
			val := promiscuous_mode == "true"
			securityPolicy.AllowPromiscuous = &val
		}
		if forged_transmits != "" {
			val := forged_transmits == "true"
			securityPolicy.ForgedTransmits = &val
		}
		if mac_changes != "" {
			val := mac_changes == "true"
			securityPolicy.MacChanges = &val
		}
	}

	// Build port group spec
	spec := types.HostPortGroupSpec{
		Name:        name,
		VswitchName: vswitch,
		VlanId:      int32(vlan),
		Policy: types.HostNetworkPolicy{
			Security: securityPolicy,
		},
	}

	err = ns.UpdatePortGroup(gc.Context(), name, spec)
	if err != nil {
		return fmt.Errorf("failed to update portgroup: %w", err)
	}

	log.Printf("[portgroupUpdate] Successfully updated portgroup %s\n", name)
	return nil
}
