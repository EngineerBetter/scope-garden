package garden

import (
	"fmt"

	"code.cloudfoundry.org/garden"
	gardenclient "code.cloudfoundry.org/garden/client"
)

type registry struct {
	client gardenclient.Client
}

func newRegistry(client gardenclient.Client) *registry {
	return &registry{
		client: client,
	}
}

func (r *registry) walkContainers(fn func(c garden.Container) error) error {
	containers, err := r.client.Containers(garden.Properties{})
	if err != nil {
		err := fmt.Errorf("error fetching containers: %v\n", err)
		return err
	}

	for _, c := range containers {
		if err := fn(c); err != nil {
			return err
		}
	}

	return nil
}
