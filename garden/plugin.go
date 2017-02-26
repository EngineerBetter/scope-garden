package garden

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	gardenclient "code.cloudfoundry.org/garden/client"
	gardenconnection "code.cloudfoundry.org/garden/client/connection"
)

type plugin struct {
	lock     sync.RWMutex
	hostname string
	registry *registry
	done     chan struct{}
	report   report
}

func NewPlugin(hostname, gardenNetwork, gardenAddr string, fetchInterval time.Duration) *plugin {
	client := gardenclient.New(
		gardenconnection.New(gardenNetwork, gardenAddr),
	)

	p := &plugin{
		hostname: hostname,
		registry: newRegistry(client),
		report:   newReport(hostname),
		done:     make(chan struct{}),
	}

	p.refreshReport(fetchInterval)

	return p
}

func (p *plugin) Close() {
	close(p.done)
}

func (p *plugin) refreshReport(interval time.Duration) {
	go func() {
		for {
			select {
			case <-time.After(interval):
				r := newReport(p.hostname)

				if err := p.registry.walkContainers(r.AddNode); err != nil {
					log.Println(err)
				}

				p.lock.Lock()
				p.report = r
				p.lock.Unlock()
			case <-p.done:
				return
			}
		}
	}()
}

func (p *plugin) Report(w http.ResponseWriter, r *http.Request) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if err := json.NewEncoder(w).Encode(p.report); err != nil {
		log.Printf("error encoding report: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
