package main

import (
	"io"
	"log"
	"net/http"
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
	http.HandleFunc("/", handler)
	log.Println("load balanceer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
