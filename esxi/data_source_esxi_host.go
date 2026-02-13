package esxi

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceEsxiHost() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceEsxiHostRead,

		Schema: map[string]*schema.Schema{
			"hostname": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ESXi host hostname or IP address.",
			},
			"version": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ESXi version and build.",
			},
			"product_name": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ESXi product name.",
			},
			"uuid": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ESXi host system UUID.",
			},
			"manufacturer": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hardware manufacturer.",
			},
			"model": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hardware model.",
			},
			"serial_number": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hardware serial number.",
			},
			"cpu_model": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CPU model.",
			},
			"cpu_packages": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of CPU packages.",
			},
			"cpu_cores": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of CPU cores.",
			},
			"cpu_threads": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of CPU threads.",
			},
			"cpu_mhz": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "CPU speed in MHz.",
			},
			"memory_size": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total memory size in MB.",
			},
			"datastores": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"capacity_gb": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"free_gb": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
				Description: "List of datastores on the ESXi host.",
			},
		},
	}
}

func dataSourceEsxiHostRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[dataSourceEsxiHostRead]")

	// Use govmomi if enabled for better host information
	if c.useGovmomi {
		return dataSourceEsxiHostReadGovmomi(d, c)
	}

	// Fallback to SSH
	return dataSourceEsxiHostReadSSH(d, c)
}

func dataSourceEsxiHostReadSSH(d *schema.ResourceData, c *Config) error {
	esxiConnInfo := getConnectionInfo(c)

	// Get basic host information
	version, productName, uuid, err := getHostInfoSSH(esxiConnInfo)
	if err != nil {
		return fmt.Errorf("Failed to get host info: %s", err)
	}

	// Get hardware information
	manufacturer, model, serialNumber, cpuModel, cpuPackages, cpuCores, cpuThreads, cpuMhz, memorySize, err := getHardwareInfoSSH(esxiConnInfo)
	if err != nil {
		return fmt.Errorf("Failed to get hardware info: %s", err)
	}

	// Get datastore information
	datastores, err := getDatastoresSSH(esxiConnInfo)
	if err != nil {
		log.Printf("[dataSourceEsxiHostRead] Warning: failed to get datastores: %s", err)
		datastores = []map[string]interface{}{}
	}

	// Set ID to hostname
	d.SetId(c.esxiHostName)

	// Set computed fields
	d.Set("hostname", c.esxiHostName)
	d.Set("version", version)
	d.Set("product_name", productName)
	d.Set("uuid", uuid)
	d.Set("manufacturer", manufacturer)
	d.Set("model", model)
	d.Set("serial_number", serialNumber)
	d.Set("cpu_model", cpuModel)
	d.Set("cpu_packages", cpuPackages)
	d.Set("cpu_cores", cpuCores)
	d.Set("cpu_threads", cpuThreads)
	d.Set("cpu_mhz", cpuMhz)
	d.Set("memory_size", memorySize)
	d.Set("datastores", datastores)

	log.Printf("[dataSourceEsxiHostRead] Successfully read ESXi host '%s'", c.esxiHostName)
	return nil
}

func getHostInfoSSH(esxiConnInfo ConnectionStruct) (string, string, string, error) {
	remote_cmd := "vmware -vl"
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get host version info")
	if err != nil {
		return "", "", "", err
	}

	lines := strings.Split(stdout, "\n")
	var version, productName, uuid string

	for _, line := range lines {
		if strings.Contains(line, "VMware ESXi") {
			productName = "VMware ESXi"
			re := regexp.MustCompile(`VMware ESXi ([0-9.]+)-([0-9]+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				version = fmt.Sprintf("%s-%s", matches[1], matches[2])
			}
		}
	}

	// Get UUID
	remote_cmd = "esxcli system uuid get"
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get host uuid")
	if err == nil {
		re := regexp.MustCompile(`UUID: (.+)`)
		matches := re.FindStringSubmatch(stdout)
		if len(matches) >= 2 {
			uuid = matches[1]
		}
	}

	return version, productName, uuid, nil
}

func getHardwareInfoSSH(esxiConnInfo ConnectionStruct) (string, string, string, string, int, int, int, int, int, error) {
	remote_cmd := "dmidecode -t system"
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get system info")
	if err != nil {
		return "", "", "", "", 0, 0, 0, 0, 0, err
	}

	var manufacturer, model, serialNumber string

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Manufacturer:") {
			manufacturer = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		if strings.Contains(line, "Product Name:") {
			model = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		if strings.Contains(line, "Serial Number:") {
			serialNumber = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

	// Get CPU info
	remote_cmd = "esxcli hardware cpu list"
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get cpu info")
	if err != nil {
		return manufacturer, model, serialNumber, "", 0, 0, 0, 0, 0, err
	}

	var cpuModel string
	var cpuPackages, cpuCores, cpuThreads, cpuMhz int

	lines = strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU Name:") {
			cpuModel = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		if strings.Contains(line, "Package Count:") {
			if val, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
				cpuPackages = val
			}
		}
		if strings.Contains(line, "Core Count:") {
			if val, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
				cpuCores = val
			}
		}
		if strings.Contains(line, "Thread Count:") {
			if val, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
				cpuThreads = val
			}
		}
		if strings.Contains(line, "Speed:") {
			speedStr := strings.TrimSpace(strings.Split(line, ":")[1])
			speedStr = strings.Replace(speedStr, "MHz", "", 1)
			if val, err := strconv.Atoi(speedStr); err == nil {
				cpuMhz = val
			}
		}
	}

	// Get memory info
	remote_cmd = "esxcli hardware memory get"
	stdout, err = runRemoteSshCommand(esxiConnInfo, remote_cmd, "get memory info")
	if err != nil {
		return manufacturer, model, serialNumber, cpuModel, cpuPackages, cpuCores, cpuThreads, cpuMhz, 0, err
	}

	var memorySize int
	lines = strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Physical Memory:") {
			memStr := strings.TrimSpace(strings.Split(line, ":")[1])
			memStr = strings.Replace(memStr, "MB", "", 1)
			if val, err := strconv.Atoi(memStr); err == nil {
				memorySize = val
			}
		}
	}

	return manufacturer, model, serialNumber, cpuModel, cpuPackages, cpuCores, cpuThreads, cpuMhz, memorySize, nil
}

func getDatastoresSSH(esxiConnInfo ConnectionStruct) ([]map[string]interface{}, error) {
	remote_cmd := "esxcli storage filesystem list"
	stdout, err := runRemoteSshCommand(esxiConnInfo, remote_cmd, "get datastores")
	if err != nil {
		return nil, err
	}

	var datastores []map[string]interface{}
	lines := strings.Split(stdout, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "VMFS") || strings.Contains(line, "NFS") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				ds := map[string]interface{}{
					"name":        fields[1],
					"type":        fields[2],
					"capacity_gb": 0, // Would need additional commands to get this
					"free_gb":     0, // Would need additional commands to get this
				}
				datastores = append(datastores, ds)
			}
		}
	}

	return datastores, nil
}