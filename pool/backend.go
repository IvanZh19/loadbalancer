package pool

import (
	"net/url"
	"sync"
)

type Backend struct {
	URL *url.URL
	alive bool
	mu sync.RWMutex
	ActiveConns int64 // for least connections
	ConsecutiveFails int64 // atomic
	ConsecutiveSuccess int64 // atomic
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.alive = alive
}

func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.alive
}
