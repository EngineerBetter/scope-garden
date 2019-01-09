package garden

import (
	"fmt"
	"math/rand"
	"time"

	"code.cloudfoundry.org/garden"
)

const (
	CFContainerNetwork = "cf-container-network"

	ContainerID               = "garden_container_id"
	ContainerPath             = "garden_container_path"
	ContainerIP               = "garden_container_ip"
	ContainerHostIP           = "garden_container_host_ip"
	ContainerExternalIP       = "garden_container_external_ip"
	ContainerState            = "garden_container_state"
	ContainerPropertiesPrefix = "garden_container_properties_"
	ContainerConcoursePrefix  = "concourse_"

	DockerContainerHostname  = "docker_container_hostname"
	DockerContainerIPsScopes = "docker_container_ips_with_scopes"
	DockerContainerIPs       = "docker_container_ips"
	DockerContainerNetworks  = "docker_container_networks"

	MemoryUsage = "garden_memory_usage"
	DiskUsage   = "garden_disk_usage"
	CPUUsage    = "garden_cpu_total_usage"
	NetworkRx   = "garden_network_rx"
	NetworkTx   = "garden_network_tx"

	ContainerAppGUID = "cf_app_guid"
)

func newReport(hostname string, appNameLookup lookupFn) report {
	return report{
		ID:                       fmt.Sprintf("%d", rand.Int63()),
		Plugins:                  []pluginSpec{pluginInfo},
		Container:                newContainer(),
		ContainerImage:           newContainerImage(),
		hostname:                 hostname,
		lookupConcourseContainer: appNameLookup,
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

	for k, v := range info.Properties {
		key := fmt.Sprintf("%s%s", ContainerPropertiesPrefix, k)
		n.Latest[key] = latest(v)
	}

	n.Sets = map[string][]string{
		DockerContainerIPsScopes: {fmt.Sprintf(";%s", info.ContainerIP)},
		DockerContainerIPs:       {info.ContainerIP},
		DockerContainerNetworks:  {CFContainerNetwork},
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

	var stepName string
	concourseContainer, found := r.lookupConcourseContainer(id)

	if !found {
		stepName = "not-found"
	}

	stepName = concourseContainer.StepName

	containerName := id
	if stepName != "not-found" {
		containerName = fmt.Sprintf("%s/%s", stepName, id[:5])
	}

	n.Latest[fmt.Sprintf("%s%s", ContainerConcoursePrefix, "build number")] = latest(concourseContainer.BuildName)
	n.Latest[fmt.Sprintf("%s%s", ContainerConcoursePrefix, "pipeline")] = latest(concourseContainer.PipelineName)
	n.Latest[fmt.Sprintf("%s%s", ContainerConcoursePrefix, "job")] = latest(concourseContainer.JobName)
	n.Latest[fmt.Sprintf("%s%s", ContainerConcoursePrefix, "type")] = latest(concourseContainer.Type)

	n.Latest["docker_container_name"] = latest(containerName)
	n.Latest["docker_image_id"] = latest(stepName)
	n.Latest[ContainerAppGUID] = latest(id)

	img := nodeSpec{
		ID:       fmt.Sprintf("%s;<container_image>", stepName),
		Topology: "container_image",
		Parents:  map[string][]string{"host": []string{host}},
		Latest: map[string]latestSpec{
			ContainerAppGUID:    latest(id),
			"docker_image_id":   latest(stepName),
			"docker_image_name": latest(stepName),
			"host_node_id":      latest(host),
		},
	}

	r.ContainerImage.Nodes[img.ID] = img

	n.Parents["container_image"] = []string{img.ID}

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
	Sets     map[string][]string   `json:"sets,omitempty"`
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
	ID                       string             `json:"ID"`
	Plugins                  []pluginSpec       `json:"Plugins"`
	Container                containerSpec      `json:"Container"`
	ContainerImage           containerImageSpec `json:"ContainerImage"`
	hostname                 string
	lookupConcourseContainer lookupFn
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
	}

	containerMetricTemplates = map[string]metricTemplateSpec{
		CPUUsage:    {ID: CPUUsage, Label: "CPU Usage", Format: "", Priority: 3},
		MemoryUsage: {ID: MemoryUsage, Label: "Memory", Format: "filesize", Priority: 4},
		DiskUsage:   {ID: DiskUsage, Label: "Disk Usage", Format: "filesize", Priority: 5},
		NetworkRx:   {ID: NetworkRx, Label: "Network RX", Format: "filesize", Priority: 6},
		NetworkTx:   {ID: NetworkTx, Label: "Network TX", Format: "filesize", Priority: 7},
	}

	containerTableTemplates = map[string]tableTemplateSpec{
		ContainerPropertiesPrefix: {ID: ContainerPropertiesPrefix, Label: "Properties", Prefix: ContainerPropertiesPrefix},
		ContainerConcoursePrefix:  {ID: ContainerConcoursePrefix, Label: "Concourse", Prefix: ContainerConcoursePrefix},
	}

	containerImageTableTemplates = map[string]tableTemplateSpec{
		ContainerConcoursePrefix: {ID: ContainerConcoursePrefix, Label: "Concourse", Prefix: ContainerConcoursePrefix},
	}

	containerImageMetadataTemplates = map[string]metadataTemplateSpec{}
)
