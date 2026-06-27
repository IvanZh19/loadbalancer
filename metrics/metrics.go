package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/IvanZh19/loadbalancer/pool"
)

type Metrics struct {
	mu sync.Mutex
	RequestCount map[string]int64 // backend URL : total req served
	ErrorCount map[string]int64 // backend URL : total errors
}

func NewMetrics(backends []*pool.Backend) *Metrics {
	return &Metrics{
		mu: sync.Mutex{},
		RequestCount: make(map[string]int64),
		ErrorCount: make(map[string]int64),
	}
}

func (m *Metrics) RecordRequest(backendURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RequestCount[backendURL]++
}

func (m *Metrics) RecordError(backendURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCount[backendURL]++
}

func (m *Metrics) Handler(pool *pool.BackendPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, b := range pool.Backends() {
			m.mu.Lock()
			url := b.URL
			reqs := m.RequestCount[url.String()]
			errs := m.ErrorCount[url.String()]
			m.mu.Unlock()
			conns := atomic.LoadInt64(&b.ActiveConns)
			fmt.Fprintf(w, "backend %s: reqs %d errs %d conns %d\n",
					url, reqs, errs, conns)
		}
	}
}
