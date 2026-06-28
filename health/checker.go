package health

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/IvanZh19/loadbalancer/pool"
)

type HealthChecker struct {
	pool *pool.BackendPool
	interval time.Duration
	timeout time.Duration
	riseThreshold int64
	fallThreshold int64
}

func NewHealthChecker (pool *pool.BackendPool, interval time.Duration, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		pool: pool,
		interval: interval,
		timeout: timeout,
		riseThreshold: 1,
		fallThreshold: 2,
	}
}

func (h *HealthChecker) updateBackendStatus(b *pool.Backend, alive bool) {
	if alive {
		atomic.AddInt64(&b.ConsecutiveSuccess, 1)
		atomic.StoreInt64(&b.ConsecutiveFails, 0)
		if atomic.LoadInt64(&b.ConsecutiveSuccess) >= h.riseThreshold && !b.IsAlive() {
			log.Printf("backend %s status changed: alive=%v", b.URL, alive)
			b.SetAlive(true)
		}
	} else {
		atomic.AddInt64(&b.ConsecutiveFails, 1)
		atomic.StoreInt64(&b.ConsecutiveSuccess, 0)
		if atomic.LoadInt64(&b.ConsecutiveFails) >= h.fallThreshold && b.IsAlive() {
			log.Printf("backend %s status changed: alive=%v", b.URL, alive)
			b.SetAlive(false)
		}
	}
}

func (h *HealthChecker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, b := range h.pool.Backends() {
					alive := h.checkBackend(b)
					h.updateBackendStatus(b, alive)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (h *HealthChecker) checkBackend(b *pool.Backend) bool {
	client := &http.Client{Timeout: h.timeout}
	resp, err := client.Get(b.URL.String() + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
