package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IvanZh19/loadbalancer/config"
	"github.com/IvanZh19/loadbalancer/health"
	"github.com/IvanZh19/loadbalancer/metrics"
	"github.com/IvanZh19/loadbalancer/pool"
	"github.com/IvanZh19/loadbalancer/proxy"
)

func parseStrategy(strategy string) pool.Strategy {
	switch strategy {
	case "least-connections":
		return &pool.LeastConnections{}
	case "ip-hash":
		return &pool.IPHash{}
	default:
		return &pool.RoundRobin{}
	}
}

func main() {
	// set up from config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	addr := cfg.Addr
	strategy := cfg.Strategy
	backends := cfg.Backends

	// health
	interval := time.Duration(cfg.Health.IntervalSecs) * time.Second
	timeout := time.Duration(cfg.Health.TimeoutSecs) * time.Second
	riseThreshold := cfg.Health.RiseThreshold
	fallThreshold := cfg.Health.FallThreshold

	p := pool.NewBackendPool(parseStrategy(strategy))
	for _, b := range backends {
		p.AddBackend(b)
	}

	m := metrics.NewMetrics(p.Backends())
	http.Handle("/metrics", m.Handler(p))
	proxy := proxy.NewProxyServer(p, m)
	http.Handle("/", proxy)

	log.Println("load balancer listening on :8080")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	checker := health.NewHealthChecker(p, interval, timeout, riseThreshold, fallThreshold)
	checker.Start(ctx)

	// the following does graceful shutdown
	server := &http.Server{Addr: addr}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
