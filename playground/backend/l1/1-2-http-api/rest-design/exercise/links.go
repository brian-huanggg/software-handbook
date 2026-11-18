//go:build exercise

// Package exercise: finish ListLinks so that
// `go test -tags exercise ./exercise/` passes.
package exercise

import (
	"net/http"
)

type Link struct {
	Code   string `json:"code"`
	Target string `json:"target"`
}

// Store holds links already sorted by code.
type Store struct {
	links []Link
}

func NewStore(links ...Link) *Store {
	return &Store{links: links}
}

func NewHandler(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /links", s.ListLinks)
	return mux
}

// ListLinks handles GET /links.
//
// TODO:
//   - ?limit=N: default 20, allowed 1 to 100, anything else is 400
//   - ?offset=N: default 0, must be 0 or more, anything else is 400
//   - ?host=H: keep only links whose target URL host is exactly H
//   - respond 200 with {"items": [...]}; no match must give [] and not null
func (s *Store) ListLinks(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
