package esxi

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func vswitchUpdate(c *Config, name string, ports int, mtu int, uplinks []string,
	link_discovery_mode string, promiscuous_mode bool, mac_changes bool, forged_transmits bool) error {
	// Use govmomi if enabled
	if c.useGovmomi {
		return vswitchUpdate_govmomi(c, name, ports, mtu, uplinks, link_discovery_mode, promiscuous_mode, mac_changes, forged_transmits)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)

	log.Println("[vswitchUpdate]")

	var foundUplinks []string
	var remote_cmd, stdout string
	var err error

	//  Set mtu and cdp
	remote_cmd = fmt.Sprintf("esxcli network vswitch standard set -m %d -c \"%s\" -v \"%s\"",
		mtu, link_discovery_mode, name)

	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "set vswitch mtu, link_discovery_mode")
	if err != nil {
		return fmt.Errorf("Failed to set vswitch mtu: %s\n%s\n", stdout, err)
	}

	//  Set security
	remote_cmd = fmt.Sprintf("esxcli network vswitch standard policy security set -f %t -m %t -p %t -v \"%s\"",
		promiscuous_mode, mac_changes, forged_transmits, name)

	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "set vswitch security")
	if err != nil {
		return fmt.Errorf("Failed to set vswitch security: %s\n%s\n", stdout, err)
	}

	//  Update uplinks
	remote_cmd = fmt.Sprintf("esxcli network vswitch standard list -v \"%s\"", name)
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "vswitch list")

	if err != nil {
		log.Printf("[vswitchUpdate] Failed to run %s: %s\n", "vswitch list", err)
		return fmt.Errorf("Failed to list vswitch: %s\n%s\n", stdout, err)
	}

	re := regexp.MustCompile(`Uplinks: (.*)`)
	foundUplinksRaw := strings.Fields(re.FindStringSubmatch(stdout)[1])
	for i, s := range foundUplinksRaw {
		foundUplinks = append(foundUplinks, strings.Replace(s, ",", "", -1))
		log.Printf("[vswitchUpdate] found uplinks[%d]: /%s/\n", i, foundUplinks[i])
	}

	//  Add uplink if needed
	for i, _ := range uplinks {
		if inArrayOfStrings(foundUplinks, uplinks[i]) == false {
			log.Printf("[vswitchUpdate] add uplinks %d (%s)\n", i, uplinks[i])
			remote_cmd = fmt.Sprintf("esxcli network vswitch standard uplink add -u \"%s\" -v \"%s\"",
				uplinks[i], name)

			stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "vswitch add uplink")
			if strings.Contains(stdout, "Not a valid pnic") {
				return fmt.Errorf("Uplink not found: %s\n", uplinks[i])
			}
			if err != nil {
				return fmt.Errorf("Failed to add vswitch uplink: %s\n%s\n", stdout, err)
			}
		}
	}

	//  Remove uplink if needed
	for _, item := range foundUplinks {
		if inArrayOfStrings(uplinks, item) == false {
			log.Printf("[vswitchUpdate] delete uplink (%s)\n", item)
			remote_cmd = fmt.Sprintf("esxcli network vswitch standard uplink remove -u \"%s\" -v \"%s\"",
				item, name)

			stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "vswitch remove uplink")
			if err != nil {
				return fmt.Errorf("Failed to remove vswitch uplink: %s\n%s\n", stdout, err)
			}
		}
	}

	return nil
}

func vswitchRead(c *Config, name string) (int, int, []string, string, bool, bool, bool, error) {
	// Use govmomi if enabled
	if c.useGovmomi {
		return vswitchRead_govmomi(c, name)
	}

	// Fallback to SSH
	esxiConnInfo := getConnectionInfo(c)
	log.Println("[vswitchRead]")

	var ports, mtu int
	var uplinks []string
	var link_discovery_mode string
	var promiscuous_mode, mac_changes, forged_transmits bool
	var remote_cmd, stdout string
	var err error

	remote_cmd = fmt.Sprintf("esxcli network vswitch standard list -v \"%s\"", name)
	stdout, _ = runRemoteSshCommand(esxiConnInfo, remote_cmd, "vswitch list")

	if stdout == "" {
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf(stdout)
	}

	re, _ := regexp.Compile(`Configured Ports: ([0-9]*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		ports, _ = strconv.Atoi(re.FindStringSubmatch(stdout)[1])
	} else {
		ports = 128
	}

	re, _ = regexp.Compile(`MTU: ([0-9]*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		mtu, _ = strconv.Atoi(re.FindStringSubmatch(stdout)[1])
	} else {
		mtu = 1500
	}

	re, _ = regexp.Compile(`CDP Status: ([a-z]*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		link_discovery_mode = re.FindStringSubmatch(stdout)[1]
	} else {
		link_discovery_mode = "listen"
	}

	re, _ = regexp.Compile(`Uplinks: (.*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		foundUplinks := strings.Fields(re.FindStringSubmatch(stdout)[1])
		log.Printf("[vswitchRead] found foundUplinks: /%s/\n", foundUplinks)
		for i, s := range foundUplinks {
			uplinks = append(uplinks, strings.Replace(s, ",", "", -1))
			log.Printf("[vswitchRead] found uplinks[%d]: /%s/\n\n\n", i, uplinks[i])
		}
	} else {
		uplinks = uplinks[:0]
	}

	remote_cmd = fmt.Sprintf("esxcli network vswitch standard policy security get -v \"%s\"", name)
	stdout, _ = runRemoteSshCommand(esxiConnInfo, remote_cmd, "vswitch policy security get")

	if stdout == "" {
		log.Printf("[vswitchRead] Failed to run %s: %s\n", "vswitch policy security get", err)
		return 0, 0, uplinks, "", false, false, false, fmt.Errorf(stdout)
	}

	re, _ = regexp.Compile(`Allow Promiscuous: (.*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		promiscuous_mode, _ = strconv.ParseBool(re.FindStringSubmatch(stdout)[1])
	} else {
		promiscuous_mode = false
	}

	re, _ = regexp.Compile(`Allow MAC Address Change: (.*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		mac_changes, _ = strconv.ParseBool(re.FindStringSubmatch(stdout)[1])
	} else {
		mac_changes = false
	}

	re, _ = regexp.Compile(`Allow Forged Transmits: (.*)`)
	if len(re.FindStringSubmatch(stdout)) > 0 {
		forged_transmits, _ = strconv.ParseBool(re.FindStringSubmatch(stdout)[1])
	} else {
		forged_transmits = false
	}

	return ports, mtu, uplinks, link_discovery_mode, promiscuous_mode,
		mac_changes, forged_transmits, nil
}

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
// Govmomi-based Network Switch Operations
// ============================================================================

// vswitchCreate_govmomi creates a vswitch using govmomi
func vswitchCreate_govmomi(c *Config, name string, ports int) error {
	log.Printf("[vswitchCreate_govmomi] Creating vswitch %s\n", name)

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

	log.Printf("[vswitchCreate_govmomi] Successfully created vswitch %s\n", name)
	return nil
}

// vswitchDelete_govmomi deletes a vswitch using govmomi
func vswitchDelete_govmomi(c *Config, name string) error {
	log.Printf("[vswitchDelete_govmomi] Deleting vswitch %s\n", name)

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

	log.Printf("[vswitchDelete_govmomi] Successfully deleted vswitch %s\n", name)
	return nil
}

// vswitchRead_govmomi reads vswitch configuration using govmomi
func vswitchRead_govmomi(c *Config, name string) (int, int, []string, string, bool, bool, bool, error) {
	log.Printf("[vswitchRead_govmomi] Reading vswitch %s\n", name)

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

	log.Printf("[vswitchRead_govmomi] Successfully read vswitch %s\n", name)
	return ports, mtu, uplinks, link_discovery_mode, promiscuous_mode, mac_changes, forged_transmits, nil
}

// vswitchUpdate_govmomi updates vswitch configuration using govmomi
func vswitchUpdate_govmomi(c *Config, name string, ports int, mtu int, uplinks []string,
	link_discovery_mode string, promiscuous_mode bool, mac_changes bool, forged_transmits bool) error {
	log.Printf("[vswitchUpdate_govmomi] Updating vswitch %s\n", name)

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

	log.Printf("[vswitchUpdate_govmomi] Successfully updated vswitch %s\n", name)
	return nil
}
