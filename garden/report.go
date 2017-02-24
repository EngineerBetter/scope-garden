package garden

import (
	"fmt"
	"math/rand"
	"time"

	"code.cloudfoundry.org/garden"
)

const (
	DockerContainerHostname   = "docker_container_hostname"
	ContainerID               = "garden_container_id"
	ContainerPath             = "garden_container_path"
	ContainerIP               = "garden_container_ip"
	ContainerHostIP           = "garden_container_host_ip"
	ContainerExternalIP       = "garden_container_external_ip"
	ContainerState            = "garden_container_state"
	ContainerEventsPrefix     = "garden_container_events_"
	ContainerPIDsPrefix       = "garden_container_pids_"
	ContainerPropertiesPrefix = "garden_container_properties_"
	ContainerPortMapPrefix    = "garden_container_portmap_"
	ContainerCFPrefix         = "cf_"

	MemoryUsage = "garden_memory_usage"
	DiskUsage   = "garden_disk_usage"
	CPUUsage    = "garden_cpu_total_usage"
	NetworkRx   = "garden_network_rx"
	NetworkTx   = "garden_network_tx"

	ContainerAppGUID   = "cf_app_guid"
	ContainerOrgGUID   = "cf_org_guid"
	ContainerSpaceGUID = "cf_space_guid"
)

func newReport(hostname string) report {
	return report{
		ID:             fmt.Sprintf("%d", rand.Int63()),
		Plugins:        []pluginSpec{pluginInfo},
		Container:      newContainer(),
		ContainerImage: newContainerImage(),
		hostname:       hostname,
	}
}

func (r *report) AddNode(c garden.Container) error {
	id := c.Handle()

	host := fmt.Sprintf("%s;<host>", r.hostname)

	n := nodeSpec{
		ID:       fmt.Sprintf("%s;<container>", id),
		Topology: "container",
		Parents:  map[string][]string{"host": []string{host}},
	}

	info, err := c.Info()
	if err != nil {
		return fmt.Errorf("error retrieving info for container %q: %v", id, err)
	}

	n.Latest = map[string]latestSpec{
		DockerContainerHostname: latest(r.hostname),
		ContainerID:             latest(id),
		ContainerPath:           latest(info.ContainerPath),
		ContainerState:          latest(info.State),
		ContainerIP:             latest(info.ContainerIP),
		ContainerHostIP:         latest(info.HostIP),
		ContainerExternalIP:     latest(info.ExternalIP),
	}

	for k, v := range info.Events {
		key := fmt.Sprintf("%s%d", ContainerEventsPrefix, k)
		n.Latest[key] = latest(v)
	}

	for k, v := range info.ProcessIDs {
		key := fmt.Sprintf("%s%d", ContainerPIDsPrefix, k)
		n.Latest[key] = latest(v)
	}

	for k, v := range info.Properties {
		key := fmt.Sprintf("%s%s", ContainerPropertiesPrefix, k)
		n.Latest[key] = latest(v)
	}

	for _, mapping := range info.MappedPorts {
		key := fmt.Sprintf("%s%d", ContainerPortMapPrefix, mapping.HostPort)
		n.Latest[key] = latest(fmt.Sprintf("%d", mapping.ContainerPort))
	}

	metrics, err := c.Metrics()
	if err != nil {
		return fmt.Errorf("error retrieving metrics for container %q: %v", id, err)
	}

	n.Metrics = map[string]metricSpec{
		CPUUsage:    metric(float64(metrics.CPUStat.Usage) / float64(time.Second)),
		MemoryUsage: metric(float64(metrics.MemoryStat.TotalUsageTowardLimit/(1024*1024)) * 1000000),
		DiskUsage:   metric(float64(metrics.DiskStat.TotalBytesUsed)),
		NetworkRx:   metric(float64(metrics.NetworkStat.RxBytes)),
		NetworkTx:   metric(float64(metrics.NetworkStat.TxBytes)),
	}

	if appGUID, err := c.Property("network.app_id"); err == nil {
		n.Latest["docker_container_name"] = latest(id)
		n.Latest["docker_image_id"] = latest(appGUID)
		n.Latest[ContainerAppGUID] = latest(appGUID)

		img := nodeSpec{
			ID:       fmt.Sprintf("%s;<container_image>", appGUID),
			Topology: "container_image",
			Parents:  map[string][]string{"host": []string{host}},
			Latest: map[string]latestSpec{
				ContainerAppGUID:    latest(appGUID),
				"docker_image_id":   latest(appGUID),
				"docker_image_name": latest(appGUID),
				"host_node_id":      latest(host),
			},
		}

		if orgGUID, err := c.Property("network.org_id"); err == nil {
			n.Latest[ContainerOrgGUID] = latest(orgGUID)
			img.Latest[ContainerOrgGUID] = latest(orgGUID)
		}

		if spaceGUID, err := c.Property("network.space_id"); err == nil {
			n.Latest[ContainerSpaceGUID] = latest(spaceGUID)
			img.Latest[ContainerSpaceGUID] = latest(spaceGUID)
		}

		r.ContainerImage.Nodes[img.ID] = img

		n.Parents["container_image"] = []string{img.ID}
	}

	r.Container.Nodes[n.ID] = n
	return nil
}

func newContainerImage() containerImageSpec {
	return containerImageSpec{
		Label:          "image",
		LabelPlural:    "images",
		Shape:          "hexagon",
		Nodes:          map[string]nodeSpec{},
		TableTemplates: containerImageTableTemplates,
	}
}

func newContainer() containerSpec {
	return containerSpec{
		Label:             "container",
		LabelPlural:       "containers",
		Shape:             "hexagon",
		MetadataTemplates: containerMetadataTemplates,
		MetricTemplates:   containerMetricTemplates,
		TableTemplates:    containerTableTemplates,
		Nodes:             map[string]nodeSpec{},
	}
}

func latest(v string) latestSpec {
	return latestSpec{
		Timestamp: time.Now(),
		Value:     v,
	}
}

func metric(value float64) metricSpec {
	now := time.Now()

	return metricSpec{
		Samples: []sampleSpec{sampleSpec{
			Timestamp: now,
			Value:     value,
		}},
		Min:   0,
		Max:   value,
		First: now,
		Last:  now,
	}
}

type nodeSpec struct {
	ID       string                `json:"id"`
	Topology string                `json:"topology,omitempty"`
	Latest   map[string]latestSpec `json:"latest,omitempty"`
	Metrics  map[string]metricSpec `json:"metrics,omitempty"`
	Parents  map[string][]string   `json:"parents,omitempty"`
}

type latestSpec struct {
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
}

type metricSpec struct {
	Samples []sampleSpec `json:"samples"`
	Min     float64      `json:"min"`
	Max     float64      `json:"max"`
	First   time.Time    `json:"first"`
	Last    time.Time    `json:"last"`
}

type sampleSpec struct {
	Timestamp time.Time `json:"date"`
	Value     float64   `json:"value"`
}

type metadataTemplateSpec struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Priority int    `json:"priority"`
	From     string `json:"from"`
	Truncate int    `json:"truncate,omitempty"`
}

type metricTemplateSpec struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Format   string `json:"format"`
	Priority int    `json:"priority"`
}

type tableTemplateSpec struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Prefix string `json:"prefix"`
}

type containerSpec struct {
	Label             string                          `json:"label"`
	LabelPlural       string                          `json:"label_plural"`
	MetadataTemplates map[string]metadataTemplateSpec `json:"metadata_templates"`
	MetricTemplates   map[string]metricTemplateSpec   `json:"metric_templates"`
	TableTemplates    map[string]tableTemplateSpec    `json:"table_templates"`
	Nodes             map[string]nodeSpec             `json:"nodes"`
	Shape             string                          `json:"shape"`
}

type containerImageSpec struct {
	Label             string                          `json:"label"`
	LabelPlural       string                          `json:"label_plural"`
	Nodes             map[string]nodeSpec             `json:"nodes"`
	Shape             string                          `json:"shape"`
	MetadataTemplates map[string]metadataTemplateSpec `json:"metadata_templates"`
	TableTemplates    map[string]tableTemplateSpec    `json:"table_templates"`
}

type pluginSpec struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Interfaces  []string `json:"interfaces"`
	APIVersion  string   `json:"api_version"`
}

type report struct {
	ID             string             `json:"ID"`
	Plugins        []pluginSpec       `json:"Plugins"`
	Container      containerSpec      `json:"Container"`
	ContainerImage containerImageSpec `json:"ContainerImage"`
	hostname       string
}

var (
	pluginInfo = pluginSpec{
		ID:          "garden",
		Label:       "garden",
		Description: "Reports on Garden containers running on the host",
		Interfaces:  []string{"reporter"},
		APIVersion:  "1",
	}

	containerMetadataTemplates = map[string]metadataTemplateSpec{
		ContainerID:         {ID: ContainerID, Label: "ID", From: "latest", Priority: 1},
		ContainerPath:       {ID: ContainerPath, Label: "Path", From: "latest", Priority: 2},
		ContainerState:      {ID: ContainerState, Label: "State", From: "latest", Priority: 3},
		ContainerIP:         {ID: ContainerIP, Label: "Container IP", From: "latest", Priority: 4},
		ContainerHostIP:     {ID: ContainerHostIP, Label: "Host IP", From: "latest", Priority: 5},
		ContainerExternalIP: {ID: ContainerExternalIP, Label: "External IP", From: "latest", Priority: 6},
		// ContainerAppGUID:    {ID: ContainerAppGUID, Label: "CF App GUID", From: "latest", Priority: 7},
		// ContainerOrgGUID:    {ID: ContainerOrgGUID, Label: "CF Org GUID", From: "latest", Priority: 8},
		// ContainerSpaceGUID:  {ID: ContainerSpaceGUID, Label: "CF Space GUID", From: "latest", Priority: 9},
	}

	containerMetricTemplates = map[string]metricTemplateSpec{
		CPUUsage:    {ID: CPUUsage, Label: "CPU Usage", Format: "", Priority: 3},
		MemoryUsage: {ID: MemoryUsage, Label: "Memory", Format: "filesize", Priority: 4},
		DiskUsage:   {ID: DiskUsage, Label: "Disk Usage", Format: "filesize", Priority: 5},
		NetworkRx:   {ID: NetworkRx, Label: "Network RX", Format: "filesize", Priority: 6},
		NetworkTx:   {ID: NetworkTx, Label: "Network TX", Format: "filesize", Priority: 7},
	}

	containerTableTemplates = map[string]tableTemplateSpec{
		ContainerEventsPrefix:     {ID: ContainerEventsPrefix, Label: "Events", Prefix: ContainerEventsPrefix},
		ContainerPIDsPrefix:       {ID: ContainerPIDsPrefix, Label: "Process IDs", Prefix: ContainerPIDsPrefix},
		ContainerPropertiesPrefix: {ID: ContainerPropertiesPrefix, Label: "Properties", Prefix: ContainerPropertiesPrefix},
		ContainerPortMapPrefix:    {ID: ContainerPortMapPrefix, Label: "Port Mapping", Prefix: ContainerPortMapPrefix},
		ContainerCFPrefix:         {ID: ContainerCFPrefix, Label: "Cloud Foundry", Prefix: ContainerCFPrefix},
	}

	containerImageTableTemplates = map[string]tableTemplateSpec{
		ContainerCFPrefix: {ID: ContainerCFPrefix, Label: "Cloud Foundry", Prefix: ContainerCFPrefix},
	}

	containerImageMetadataTemplates = map[string]metadataTemplateSpec{
	// ContainerAppGUID:   {ID: ContainerAppGUID, Label: "CF App GUID", Priority: 1, From: "latest"},
	// ContainerOrgGUID:   {ID: ContainerOrgGUID, Label: "CF Org GUID", Priority: 2, From: "latest"},
	// ContainerSpaceGUID: {ID: ContainerSpaceGUID, Label: "CF Space GUID", Priority: 3, From: "latest"},
	}
)
