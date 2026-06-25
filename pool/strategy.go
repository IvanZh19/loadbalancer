package pool

import (
	"hash/fnv"
	"net/http"
	"sync/atomic"
)

type Strategy interface {
	Pick(backends []*Backend, r *http.Request) *Backend
}

func aliveBackends(backends []*Backend) []*Backend {
	var alive []*Backend
	for _, b := range backends {
		if b.IsAlive() {
			alive = append(alive, b)
		}
	}
	return alive
}

type RoundRobin struct {
	counter uint64
}

func (rr *RoundRobin) Pick(backends []*Backend, r *http.Request) *Backend {
	alive := aliveBackends(backends)
	if len(alive) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&rr.counter, 1) % uint64(len(alive))
	return alive[idx]
}

type LeastConnections struct{}

func (lc *LeastConnections) Pick(backends []*Backend, r *http.Request) *Backend {
	alive := aliveBackends(backends)
	if len(alive) == 0 {
		return nil
	}
	best := alive[0]
	for _, b := range alive[1:] {
		if b.ActiveConns < best.ActiveConns {
			best = b
		}
	}
	return best
}

type IPHash struct{}

func (ih *IPHash) Pick(backends []*Backend, r *http.Request) *Backend {
	alive := aliveBackends(backends)
	if len(alive) == 0 {
		return nil
	}
	h := fnv.New32a()
	h.Write([]byte(r.RemoteAddr))
	idx := h.Sum32() % uint32(len(alive))
	return alive[idx]
}
