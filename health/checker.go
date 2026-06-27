package health

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/IvanZh19/loadbalancer/pool"
)

type HealthChecker struct {
	pool *pool.BackendPool
	interval time.Duration
	timeout time.Duration
}

func NewHealthChecker (pool *pool.BackendPool, interval time.Duration, timeout time.Duration) *HealthChecker {
	return &HealthChecker{pool: pool, interval: interval, timeout: timeout}
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
					if alive != b.IsAlive() {
						log.Printf("backend %s status changed: alive=%v", b.URL, alive)
					}
					b.SetAlive(alive)
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
