//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/api.go` to compile-check only.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu    sync.Mutex
	links = map[string]string{}
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", redirect)
	mux.HandleFunc("POST /links/{code}", create)
	mux.HandleFunc("GET /links/{code}/delete", remove)
	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func reply(w http.ResponseWriter, v map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	target, ok := links[r.PathValue("code")]
	mu.Unlock()
	if !ok {
		reply(w, map[string]any{"ok": false, "error": "not found"})
		return
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func create(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+os.Getenv("API_TOKEN") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2048))
	if err != nil {
		reply(w, map[string]any{"ok": false, "error": "bad body"})
		return
	}
	mu.Lock()
	links[r.PathValue("code")] = strings.TrimSpace(string(body))
	mu.Unlock()
	reply(w, map[string]any{"ok": true})
}

func remove(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	delete(links, r.PathValue("code"))
	mu.Unlock()
	reply(w, map[string]any{"ok": true})
}
