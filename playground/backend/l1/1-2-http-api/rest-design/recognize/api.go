//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/api.go` to compile-check only.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	mux.HandleFunc("POST /createLink", createLink)
	mux.HandleFunc("GET /getLink", getLink)
	mux.HandleFunc("POST /link/{id}/remove", removeLink)
	mux.HandleFunc("GET /link/search/{keyword}", searchLinks)
	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("reply: %v", err)
	}
}

func createLink(w http.ResponseWriter, r *http.Request) {
	var in struct{ Target string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil || in.Target == "" {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	mu.Lock()
	id := strconv.Itoa(len(links) + 1)
	links[id] = in.Target
	mu.Unlock()
	reply(w, http.StatusOK, map[string]string{"id": id})
}

func getLink(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	target, ok := links[r.URL.Query().Get("id")]
	mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	reply(w, http.StatusOK, map[string]string{"target": target})
}

func removeLink(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	_, ok := links[r.PathValue("id")]
	delete(links, r.PathValue("id"))
	mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func searchLinks(w http.ResponseWriter, r *http.Request) {
	kw := r.PathValue("keyword")
	found := map[string]string{}
	mu.Lock()
	for id, target := range links {
		if strings.Contains(target, kw) {
			found[id] = target
		}
	}
	mu.Unlock()
	reply(w, http.StatusOK, found)
}
