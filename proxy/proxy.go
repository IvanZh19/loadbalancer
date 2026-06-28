package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/IvanZh19/loadbalancer/metrics"
	"github.com/IvanZh19/loadbalancer/pool"
)

type ProxyServer struct {
	pool *pool.BackendPool
	client *http.Client
	metrics *metrics.Metrics
}

func NewProxyServer(pool *pool.BackendPool, m *metrics.Metrics) *ProxyServer {
	transport := &http.Transport{
		MaxIdleConns: 30, // comfortable total among all hosts
		MaxIdleConnsPerHost: 10, // more realistic amount to keep warm
		IdleConnTimeout: time.Minute, // slightly aggressive but fine
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
			KeepAlive: 20 * time.Second, // slightly more responsive
		}).DialContext,
	}
	return &ProxyServer{
		pool: pool,
		client: &http.Client{Transport: transport},
		metrics: m,
	}
}

func isIdempotent(method string) bool {
	// empty string means GET for client requests
	idempotent := []string{"", "GET", "HEAD", "OPTIONS", "PUT", "DELETE"}
	return slices.Contains(idempotent, method)
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	maxRetries := min(3, len(p.pool.Backends()))
	for attempt := 0; attempt < maxRetries; attempt++ {
		backend := p.pool.NextBackend(r)
		if backend == nil {
			// just return, as we will get no alive backends
			http.Error(w, "no backends available", http.StatusServiceUnavailable)
			return
		}

		req, err := http.NewRequest(r.Method, backend.URL.String()+r.URL.Path, r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusInternalServerError)
			p.metrics.RecordError(backend.URL.String())
			// no retry here since r.Body is consumed
			return
		}
		req.Header = r.Header.Clone()
		for _, h := range []string{"Connection", "Transfer-Encoding", "Te", "Trailers", "Upgrade"} {
			req.Header.Del(h)
		}

		atomic.AddInt64(&backend.ActiveConns, 1)
		start := time.Now()
		resp, err := p.client.Do(req)
		duration := time.Since(start)
		p.metrics.RecordLatency(backend.URL.String(), duration)
		if err != nil {
			atomic.AddInt64(&backend.ActiveConns, -1)
			p.metrics.RecordError(backend.URL.String())
			if isIdempotent(r.Method) {
				continue
			}
			http.Error(w, "backend error", http.StatusBadGateway)
			return
		}
		defer atomic.AddInt64(&backend.ActiveConns, -1)
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		p.metrics.RecordRequest(backend.URL.String())
		log.Printf("backend %s: %s %s -> %d (%v)",
				backend.URL, r.Method, r.URL.Path, resp.StatusCode, duration)
		return
	}
	http.Error(w, "all backends failed", http.StatusBadGateway)
}
