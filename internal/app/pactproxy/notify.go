package pactproxy

import (
	"sync"
	"time"
)

type notify struct {
	notify chan struct{}
	mu     sync.Mutex
}

func NewNotify() *notify {
	return &notify{
		notify: make(chan struct{}),
		mu:     sync.Mutex{},
	}
}

func (n *notify) Wait(timeout time.Duration) {
	n.mu.Lock()
	notify := n.notify
	n.mu.Unlock()
	for {
		select {
		case <-notify:
			return
		case <-time.After(timeout):
			return
		}
	}
}

func (n *notify) Notify() {
	n.mu.Lock()
	close(n.notify)
	n.notify = make(chan struct{})
	n.mu.Unlock()
}
