//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/handlers.go` to compile-check only.
package main

import (
	"html/template"
	"net/http"
)

type Server struct{}

func (s *Server) deleteLink(slug string) {}

func (s *Server) lookup(slug string) error { return nil }

var page = template.Must(template.New("p").Parse("<p>備註：{{.}}</p>"))

func routes(s *Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /report", func(w http.ResponseWriter, r *http.Request) {
		note := r.URL.Query().Get("note")
		page.Execute(w, template.HTML(note))
	})
	mux.HandleFunc("GET /links/delete", func(w http.ResponseWriter, r *http.Request) {
		s.deleteLink(r.URL.Query().Get("slug"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /links/lookup", func(w http.ResponseWriter, r *http.Request) {
		if err := s.lookup(r.URL.Query().Get("slug")); err != nil {
			http.Error(w, "lookup failed: "+err.Error(), http.StatusInternalServerError)
		}
	})
	return http.NewCrossOriginProtection().Handler(mux)
}
