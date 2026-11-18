package main

import (
	"net/http"
	"time"
)

// NewHandler maps resources to URLs: /links is the collection,
// /links/{code} is one item, and the method says what to do with it.
func NewHandler(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", s.Redirect)
	mux.HandleFunc("GET /links", s.ListLinks)
	mux.HandleFunc("POST /links", s.CreateLink)
	mux.HandleFunc("GET /links/{code}", s.ShowLink)
	mux.HandleFunc("PUT /links/{code}", s.PutLink)
	mux.HandleFunc("DELETE /links/{code}", s.DeleteLink)
	mux.HandleFunc("POST /links/{code}/disable", s.DisableLink)
	mux.HandleFunc("GET /users/{owner}/links", s.ListOwnerLinks)
	return mux
}

func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
