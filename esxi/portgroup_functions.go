package esxi

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/jszwec/csvutil"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type portgroupSecurityPolicy struct {
	AllowForgedTransmits  bool `csv:"AllowForgedTransmits"`
	AllowMACAddressChange bool `csv:"AllowMACAddressChange"`
	AllowPromiscuous      bool `csv:"AllowPromiscuous"`
}

func portgroupRead(c *Config, name string) (string, int, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return portgroupRead_govmomi(c, name)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Println("[portgroupRead]")

	var vswitch string
	var vlan int
	var err error

	//  get portgroup info
	remote_cmd := fmt.Sprintf("esxcli network vswitch standard portgroup list |grep -m 1 \"^%s  \"", name)

	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "portgroup list")
	if stdout == "" {
		return "", 0, fmt.Errorf("Failed to list portgroup: %s\n%s\n", stdout, err)
	}

	re, _ := regexp.Compile("(  .*  )  +[0-9]+  +[0-9]+$")
	if len(re.FindStringSubmatch(stdout)) > 0 {
		vswitch = strings.Trim(re.FindStringSubmatch(stdout)[1], " ")
	} else {
		vswitch = ""
	}

	re, _ = regexp.Compile("  +([0-9]+)$")
	if len(re.FindStringSubmatch(stdout)) > 0 {
		vlan, _ = strconv.Atoi(re.FindStringSubmatch(stdout)[1])
	} else {
		vlan = 0
	}

	return vswitch, vlan, nil
}

func portgroupSecurityPolicyRead(c *Config, name string) (*portgroupSecurityPolicy, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return portgroupSecurityPolicyRead_govmomi(c, name)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)

	remote_cmd := fmt.Sprintf("esxcli --formatter=csv network vswitch standard portgroup policy security get -p \"%s\"", name)
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "portgroup security policy")
	if stdout == "" {
		return nil, fmt.Errorf("Failed to get the portgroup security policy: %s\n%s\n", stdout, err)
	}

	var policies []portgroupSecurityPolicy
	if err = csvutil.Unmarshal([]byte(stdout), &policies); err != nil || len(policies) != 1 {
		return nil, fmt.Errorf("Failed to parse the portgroup security policy: %s\n%s\n", stdout, err)
	}

	return &policies[0], nil
}

// ============================================================================
// Govmomi-based Port Group Operations
// ============================================================================

// portgroupCreate_govmomi creates a port group using govmomi
func portgroupCreate_govmomi(c *Config, name string, vswitch string) error {
	log.Printf("[portgroupCreate_govmomi] Creating portgroup %s on vswitch %s\n", name, vswitch)

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

	log.Printf("[portgroupCreate_govmomi] Successfully created portgroup %s\n", name)
	return nil
}

// portgroupDelete_govmomi deletes a port group using govmomi
func portgroupDelete_govmomi(c *Config, name string) error {
	log.Printf("[portgroupDelete_govmomi] Deleting portgroup %s\n", name)

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

	log.Printf("[portgroupDelete_govmomi] Successfully deleted portgroup %s\n", name)
	return nil
}

// portgroupRead_govmomi reads port group configuration using govmomi
func portgroupRead_govmomi(c *Config, name string) (string, int, error) {
	log.Printf("[portgroupRead_govmomi] Reading portgroup %s\n", name)

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

	log.Printf("[portgroupRead_govmomi] Successfully read portgroup %s\n", name)
	return vswitch, vlan, nil
}

// portgroupSecurityPolicyRead_govmomi reads port group security policy using govmomi
func portgroupSecurityPolicyRead_govmomi(c *Config, name string) (*portgroupSecurityPolicy, error) {
	log.Printf("[portgroupSecurityPolicyRead_govmomi] Reading security policy for portgroup %s\n", name)

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

	log.Printf("[portgroupSecurityPolicyRead_govmomi] Successfully read security policy\n")
	return policy, nil
}

// portgroupUpdate_govmomi updates port group configuration using govmomi
func portgroupUpdate_govmomi(c *Config, name string, vlan int, promiscuous_mode, forged_transmits, mac_changes string) error {
	log.Printf("[portgroupUpdate_govmomi] Updating portgroup %s\n", name)

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
	vswitch, _, err := portgroupRead_govmomi(c, name)
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

	log.Printf("[portgroupUpdate_govmomi] Successfully updated portgroup %s\n", name)
	return nil
}
