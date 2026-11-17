//go:build exercise

// Package exercise: finish PutLink and DeleteLink so that
// `go test -tags exercise ./exercise/` passes.
package exercise

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

type Store struct {
	mu    sync.Mutex
	links map[string]string
}

func NewStore() *Store {
	return &Store{links: make(map[string]string)}
}

func NewHandler(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /links/{code}", s.PutLink)
	mux.HandleFunc("DELETE /links/{code}", s.DeleteLink)
	return mux
}

// PutLink handles PUT /links/{code}. The body is the target URL.
//
// TODO:
//   - empty body: 400
//   - new code: 201 with Location: /links/{code}
//   - existing code: replace the target, 204
func (s *Store) PutLink(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2048))
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	target := strings.TrimSpace(string(body))
	s.mu.Lock()
	s.links[r.PathValue("code")] = target
	s.mu.Unlock()
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// DeleteLink handles DELETE /links/{code}.
//
// TODO:
//   - existing code: remove it, 204
//   - unknown code: 404
func (s *Store) DeleteLink(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
