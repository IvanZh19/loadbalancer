package pool

import (
	"net/http"
	"net/url"
)

type BackendPool struct {
	backends []*Backend
	strategy Strategy
}

func NewBackendPool(strategy Strategy) *BackendPool {
	return &BackendPool{strategy: strategy}
}

func (p *BackendPool) AddBackend(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	p.backends = append(p.backends, &Backend{URL: u, alive: true})
	return nil
}

func (p *BackendPool) NextBackend(r *http.Request) *Backend {
	return p.strategy.Pick(p.backends, r)
}

// mainly for health checking later
func (p *BackendPool) Backends() []*Backend {
	return p.backends
}
