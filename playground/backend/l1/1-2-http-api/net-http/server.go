package main

import (
	"log"
	"net/http"
	"time"
)

// NewHandler builds its own ServeMux instead of using http.DefaultServeMux,
// so no imported package can register routes behind our back.
func NewHandler(store *Store, token string, logger *log.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", store.Redirect)
	mux.Handle("PUT /links/{code}", Chain(
		http.HandlerFunc(store.Create),
		RequireToken(token),
	))

	// Logging is outermost: it must see the 500 that Recover writes,
	// otherwise a panicking request would leave no access-log line.
	return Chain(mux, Logging(logger), Recover(logger))
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
