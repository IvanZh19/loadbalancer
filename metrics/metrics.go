package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IvanZh19/loadbalancer/pool"
)

type Metrics struct {
	mu sync.Mutex
	RequestCount map[string]int64 // backend URL : total req served
	ErrorCount map[string]int64 // backend URL : total errors
	TotalLatency map[string]int64 // backend URL : cumulative microseconds
}

func NewMetrics(backends []*pool.Backend) *Metrics {
	return &Metrics{
		mu: sync.Mutex{},
		RequestCount: make(map[string]int64),
		ErrorCount: make(map[string]int64),
		TotalLatency: make(map[string]int64),
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

func (m *Metrics) RecordLatency(backendURL string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalLatency[backendURL] += d.Microseconds()
}

func (m *Metrics) Handler(pool *pool.BackendPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, b := range pool.Backends() {
			m.mu.Lock()
			url := b.URL
			reqs := m.RequestCount[url.String()]
			errs := m.ErrorCount[url.String()]
			var avgLatency int64
			if reqs > 0 {
				avgLatency = m.TotalLatency[url.String()] / reqs
			}
			m.mu.Unlock()
			conns := atomic.LoadInt64(&b.ActiveConns)
			fmt.Fprintf(w, "backend %s: reqs %d errs %d activeconns %d avglatency %dus\n",
					url, reqs, errs, conns, avgLatency)
		}
	}
}
