package main

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

// Store keeps short codes in memory. The mutex is needed because every
// request runs in its own goroutine and they all share this map.
type Store struct {
	mu    sync.RWMutex
	links map[string]string
}

func NewStore() *Store {
	return &Store{links: make(map[string]string)}
}

func (s *Store) Get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.links[code]
	return target, ok
}

func (s *Store) Put(code, target string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links[code] = target
}

// Redirect handles GET /{code}.
func (s *Store) Redirect(w http.ResponseWriter, r *http.Request) {
	target, ok := s.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// Create handles PUT /links/{code}; the request body is the target URL.
func (s *Store) Create(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2048))
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	target := strings.TrimSpace(string(body))
	if target == "" {
		http.Error(w, "empty target", http.StatusBadRequest)
		return
	}
	s.Put(r.PathValue("code"), target)
	w.WriteHeader(http.StatusCreated)
}
