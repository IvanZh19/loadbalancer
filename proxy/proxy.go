package proxy

import (
	"io"
	"net/http"
	"sync/atomic"

	"github.com/IvanZh19/loadbalancer/pool"
)

type ProxyServer struct {
	pool *pool.BackendPool
	client *http.Client
}

func NewProxyServer(pool *pool.BackendPool) *ProxyServer {
	return &ProxyServer{pool: pool, client: &http.Client{}}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := p.pool.NextBackend(r)
	if backend == nil {
		http.Error(w, "no backends available", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequest(r.Method, backend.URL.String()+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()
	for _, h := range []string{"Connection", "Transfer-Encoding", "Te", "Trailers", "Upgrade"} {
		req.Header.Del(h)
	}

	atomic.AddInt64(&backend.ActiveConns, 1)
	defer atomic.AddInt64(&backend.ActiveConns, -1)
	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
