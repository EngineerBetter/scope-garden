package garden

import (
	"log"

	"code.cloudfoundry.org/garden"
	gardenclient "code.cloudfoundry.org/garden/client"
)

type registry struct {
	client gardenclient.Client
	hostID string
}

func newRegistry(client gardenclient.Client, hostID string) *registry {
	return &registry{
		client: client,
		hostID: hostID,
	}
}

func (r *registry) walkContainers(fn func(c container)) {
	containers, err := r.client.Containers(garden.Properties{})
	if err != nil {
		log.Printf("error fetching containers: %v\n", err)
		return
	}

	for _, c := range containers {
		fn(newContainer(c, r.hostID))
	}
}
