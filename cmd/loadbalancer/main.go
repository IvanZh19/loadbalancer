package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/IvanZh19/loadbalancer/health"
	"github.com/IvanZh19/loadbalancer/pool"
)

const backendURL = "http://localhost:9001"

func handler(w http.ResponseWriter, r *http.Request) {
	// build new request to the backend, copy method, path, body
	req, err := http.NewRequest(r.Method, backendURL+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()

	// send
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// copy resp headers, status, body back to client
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	p := pool.NewBackendPool(&pool.RoundRobin{})
	p.AddBackend("http://localhost:9001")
	p.AddBackend("http://localhost:9002")
	p.AddBackend("http://localhost:9003")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend := p.NextBackend(r)
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

		resp, err := http.DefaultClient.Do(req)
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
	})

	log.Println("load balancer listening on :8080")
	ctx := context.Background()
	checker := health.NewHealthChecker(p, 10 * time.Second, 2 * time.Second)
	checker.Start(ctx)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
