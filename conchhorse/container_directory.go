package conchhorse

import (
	"log"
	"sync"
	"time"

	"github.com/concourse/atc"
	"github.com/concourse/go-concourse/concourse"
)

type directory struct {
	lock       sync.RWMutex
	done       chan struct{}
	containers map[string]atc.Container
	client     concourse.Client
}

func NewAppDirectory(client concourse.Client, fetchInterval time.Duration) *directory {
	d := &directory{
		client: client,
		done:   make(chan struct{}),
	}

	d.fetch(fetchInterval)

	return d
}

func (d *directory) Close() {
	d.lock.Lock()
	defer d.lock.Unlock()

	close(d.done)
}

func (d *directory) ConcourseContainer(guid string) (atc.Container, bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	if container, found := d.containers[guid]; found {
		return container, true
	}

	return atc.Container{}, false
}

func (d *directory) set(containers []atc.Container) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.containers = map[string]atc.Container{}

	for _, container := range containers {
		log.Printf("found some containers: %+v\n", containers)
		d.containers[container.ID] = container
	}
}

func (d *directory) fetch(interval time.Duration) {
	go func() {
		for {
			select {
			case <-time.After(interval):
				team := d.client.Team("main")
				containers, err := team.ListContainers(map[string]string{})
				if err != nil {
					log.Printf("error fetching Concourse containers: %v\n", err)
					continue
				}

				d.set(containers)
			case <-d.done:
				return
			}
		}
	}()
}
