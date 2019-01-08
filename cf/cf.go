package cf

import (
	"sync"
)

type directory struct {
	lock sync.RWMutex
	done chan struct{}
}

func NewAppDirectory() *directory {
	d := &directory{
		done: make(chan struct{}),
	}

	return d
}

func (d *directory) Close() {
	d.lock.Lock()
	defer d.lock.Unlock()

	close(d.done)
}

func (d *directory) AppName(guid string) (string, bool) {
	return "guid", false
}
