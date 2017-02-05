package garden

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	gardenclient "code.cloudfoundry.org/garden/client"
	gardenconnection "code.cloudfoundry.org/garden/client/connection"
)

type plugin struct {
	lock     sync.RWMutex
	hostname string
	registry *registry
}

func NewPlugin(hostname, gardenNetwork, gardenAddr string) *plugin {
	client := gardenclient.New(
		gardenconnection.New(gardenNetwork, gardenAddr),
	)

	return &plugin{
		hostname: hostname,
		registry: newRegistry(client),
	}
}

func (p *plugin) Report(w http.ResponseWriter, r *http.Request) {
	p.lock.Lock()
	defer p.lock.Unlock()

	report := newReport(p.hostname)

	if err := p.registry.walkContainers(report.AddNode); err != nil {
		log.Println(err)
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		log.Printf("error encoding report: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
