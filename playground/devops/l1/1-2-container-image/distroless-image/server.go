package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sync"
)

// store keeps links in memory: this example is about the image, not storage.
type store struct {
	mu    sync.RWMutex
	links map[string]string
}

func newHandler() http.Handler {
	s := &store{links: make(map[string]string)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /links", s.create)
	mux.HandleFunc("GET /{code}", s.redirect)
	return mux
}

func (s *store) create(w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("url")
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "url must be an absolute http(s) URL", http.StatusBadRequest)
		return
	}
	sum := sha256.Sum256([]byte(target))
	code := hex.EncodeToString(sum[:])[:6]

	s.mu.Lock()
	s.links[code] = target
	s.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(code + "\n"))
}

func (s *store) redirect(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	target, ok := s.links[r.PathValue("code")]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
