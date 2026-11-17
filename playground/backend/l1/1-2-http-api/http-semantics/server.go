package main

import (
	"net/http"
	"time"
)

// NewHandler registers one route per method and path. The mux answers
// 405 with an Allow header for a known path with an unknown method.
func NewHandler(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", s.Redirect)
	mux.HandleFunc("POST /links", s.PostLink)
	mux.HandleFunc("GET /links/{code}", s.Show)
	mux.HandleFunc("PUT /links/{code}", s.PutLink)
	mux.HandleFunc("DELETE /links/{code}", s.DeleteLink)
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
