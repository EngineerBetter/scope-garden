package cf

import (
	"log"
	"sync"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type directory struct {
	lock   sync.RWMutex
	client *cfclient.Client
	apps   map[string]cfclient.App
	done   chan struct{}
}

func NewAppDirectory(addr, username, password, clientID, clientSecret string, skipSSLValidation bool, fetchInterval time.Duration) *directory {
	if addr == "" {
		log.Println("Cloud Foundry API URL not set, skipping app lookup")
		return &directory{}
	}

	c, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        addr,
		Username:          username,
		Password:          password,
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		SkipSslValidation: skipSSLValidation,
	})

	if err != nil {
		log.Printf("error instantiating CF client: %v\n", err)
		return &directory{}
	}

	d := &directory{
		client: c,
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

func (d *directory) AppName(guid string) (string, bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	if app, found := d.apps[guid]; found {
		return app.Name, true
	}

	return "", false
}

func (d *directory) set(apps []cfclient.App) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.apps = map[string]cfclient.App{}

	for _, app := range apps {
		d.apps[app.Guid] = app
	}
}

func (d *directory) fetch(interval time.Duration) {
	go func() {
		for {
			select {
			case <-time.After(interval):
				apps, err := d.client.ListApps()
				if err != nil {
					log.Printf("error fetching CF apps: %v\n", err)
					continue
				}

				d.set(apps)
			case <-d.done:
				return
			}
		}
	}()
}
