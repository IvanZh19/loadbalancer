package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IvanZh19/loadbalancer/health"
	"github.com/IvanZh19/loadbalancer/metrics"
	"github.com/IvanZh19/loadbalancer/pool"
	"github.com/IvanZh19/loadbalancer/proxy"
)

func main() {
	p := pool.NewBackendPool(&pool.RoundRobin{})
	p.AddBackend("http://localhost:9001")
	p.AddBackend("http://localhost:9002")
	p.AddBackend("http://localhost:9003")

	m := metrics.NewMetrics(p.Backends())
	http.Handle("/metrics", m.Handler(p))
	proxy := proxy.NewProxyServer(p, m)
	http.Handle("/", proxy)

	log.Println("load balancer listening on :8080")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	checker := health.NewHealthChecker(p, 10 * time.Second, 2 * time.Second)
	checker.Start(ctx)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
