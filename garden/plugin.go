package garden

import (
	"log"
	"net/http"
	"sync"

	gardenclient "code.cloudfoundry.org/garden/client"
	gardenconnection "code.cloudfoundry.org/garden/client/connection"
	"github.com/ugorji/go/codec"
	scopexfer "github.com/weaveworks/scope/common/xfer"
	scopereport "github.com/weaveworks/scope/report"
)

const (
	ImageID          = "garden_image_id"
	ImageName        = "garden_image_name"
	ImageSize        = "garden_image_size"
	ImageVirtualSize = "garden_image_virtual_size"
	ImageLabelPrefix = "garden_image_label_"
	IsInHostNetwork  = "garden_is_in_host_network"
	ImageTableID     = "image_table"

	ContainerName          = "garden_container_name"
	ContainerCommand       = "garden_container_command"
	ContainerPorts         = "garden_container_ports"
	ContainerCreated       = "garden_container_created"
	ContainerNetworks      = "garden_container_networks"
	ContainerIPs           = "garden_container_ips"
	ContainerHostname      = "garden_container_hostname"
	ContainerIPsWithScopes = "garden_container_ips_with_scopes"
	ContainerState         = "garden_container_state"
	ContainerStateHuman    = "garden_container_state_human"
	ContainerUptime        = "garden_container_uptime"
	ContainerRestartCount  = "garden_container_restart_count"
	ContainerNetworkMode   = "garden_container_network_mode"

	MemoryMaxUsage = "garden_memory_max_usage"
	MemoryUsage    = "garden_memory_usage"
	MemoryFailcnt  = "garden_memory_failcnt"
	MemoryLimit    = "garden_memory_limit"

	CPUPercpuUsage       = "garden_cpu_per_cpu_usage"
	CPUUsageInUsermode   = "garden_cpu_usage_in_usermode"
	CPUTotalUsage        = "garden_cpu_total_usage"
	CPUUsageInKernelmode = "garden_cpu_usage_in_kernelmode"
	CPUSystemCPUUsage    = "garden_cpu_system_cpu_usage"

	LabelPrefix = "garden_label_"
	EnvPrefix   = "garden_env_"

	ContainerID = "garden_container_id"
)

var spec = scopexfer.PluginSpec{
	ID:          "garden",
	Label:       "garden",
	Description: "reports on Garden containers running on the host",
	Interfaces:  []string{"reporter"},
	APIVersion:  "1",
}

var (
	ContainerMetadataTemplates = scopereport.MetadataTemplates{
		ContainerCommand:    {ID: ContainerCommand, Label: "Command", From: scopereport.FromLatest, Priority: 1},
		ContainerStateHuman: {ID: ContainerStateHuman, Label: "State", From: scopereport.FromLatest, Priority: 2},
		ContainerNetworks:   {ID: ContainerNetworks, Label: "Networks", From: scopereport.FromSets, Priority: 3},
		ContainerIPs:        {ID: ContainerIPs, Label: "IPs", From: scopereport.FromSets, Priority: 4},
		ContainerPorts:      {ID: ContainerPorts, Label: "Ports", From: scopereport.FromSets, Priority: 5},
		ContainerID:         {ID: ContainerID, Label: "ID", From: scopereport.FromLatest, Truncate: 12, Priority: 6},
	}

	ContainerMetricTemplates = scopereport.MetricTemplates{
		CPUTotalUsage: {ID: CPUTotalUsage, Label: "CPU", Format: scopereport.PercentFormat, Priority: 1},
		MemoryUsage:   {ID: MemoryUsage, Label: "Memory", Format: scopereport.FilesizeFormat, Priority: 2},
	}
)

type plugin struct {
	lock     sync.RWMutex
	hostID   string
	registry *registry
}

func NewPlugin(hostID, gardenNetwork, gardenAddr string) *plugin {
	client := gardenclient.New(
		gardenconnection.New(gardenNetwork, gardenAddr),
	)

	return &plugin{
		hostID:   hostID,
		registry: newRegistry(client, hostID),
	}
}

func (p *plugin) Report(w http.ResponseWriter, r *http.Request) {
	p.lock.Lock()
	defer p.lock.Unlock()

	report, err := p.makeReport()
	if err != nil {
		log.Printf("error making report: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	report.Container = report.Container.Merge(containerTopology())

	nodes := []scopereport.Node{}
	p.registry.walkContainers(func(c container) {
		nodes = append(nodes, c.node())
	})

	for _, node := range nodes {
		report.Container.AddNode(node)
	}

	if err := codec.NewEncoder(w, &codec.JsonHandle{}).Encode(report); err != nil {
		// if err := report.WriteBinary(w, gzip.BestCompression); err != nil {
		log.Printf("error encoding report: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (p *plugin) makeReport() (*scopereport.Report, error) {
	r := scopereport.MakeReport()

	r.Plugins = r.Plugins.Add(spec)

	return &r, nil
}

func containerTopology() scopereport.Topology {
	return scopereport.MakeTopology().
		WithMetadataTemplates(ContainerMetadataTemplates).
		WithMetricTemplates(ContainerMetricTemplates)
}
