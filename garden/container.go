package garden

import (
	"log"
	"sync"
	"time"

	"code.cloudfoundry.org/garden"
	scopereport "github.com/weaveworks/scope/report"
)

type container struct {
	sync.RWMutex
	hostID     string
	gcontainer garden.Container
}
type ContainerInfo struct {
	State         string   // Either "active" or "stopped".
	Events        []string // List of events that occurred for the container. It currently includes only "oom" (Out Of Memory) event if it occurred.
	HostIP        string   // The IP address of the gateway which controls the host side of the container's virtual ethernet pair.
	ContainerIP   string   // The IP address of the container side of the container's virtual ethernet pair.
	ExternalIP    string   //
	ContainerPath string   // The path to the directory holding the container's files (both its control scripts and filesystem).
	ProcessIDs    []string // List of running processes.
}

func newContainer(c garden.Container, hostID string) container {
	return container{
		hostID:     hostID,
		gcontainer: c,
	}
}

func (c container) node() scopereport.Node {
	c.RLock()
	defer c.RUnlock()

	var (
		id   = c.gcontainer.Handle()
		node = scopereport.MakeNode(scopereport.MakeContainerNodeID(id))
	)

	info, err := c.gcontainer.Info()
	if err != nil {
		log.Printf("error retrieving info for container %q: %v\n", id, err)
		return node
	}

	node = node.WithLatests(
		map[string]string{
			ContainerID:       id,
			ContainerName:     "garden",
			ContainerCommand:  info.ContainerPath,
			ContainerHostname: info.ContainerIP,
			ContainerState:    info.State,
		},
	).WithParents(
		scopereport.EmptySets,
	).AddPrefixTable(
		LabelPrefix,
		info.Properties,
	)

	metrics, err := c.gcontainer.Metrics()
	if err != nil {
		log.Printf("error retrieving metrics for container %q: %v\n", id, err)
		return node
	}

	return node.WithMetrics(
		scopereport.Metrics{
			MemoryUsage: scopereport.MakeMetric(
				makeSamples(metrics.MemoryStat.TotalUsageTowardLimit),
			),
			CPUTotalUsage: scopereport.MakeMetric(
				makeSamples(metrics.CPUStat.User),
			),
		},
	)
}

func makeSamples(value uint64) []scopereport.Sample {
	return []scopereport.Sample{
		scopereport.Sample{
			Timestamp: time.Now(),
			Value:     float64(value),
		},
	}
}
