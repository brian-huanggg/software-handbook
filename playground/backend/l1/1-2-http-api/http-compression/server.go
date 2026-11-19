package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"
)

type link struct {
	Code   string `json:"code"`
	Target string `json:"target"`
	Owner  string `json:"owner"`
}

// NewHandler builds the URL shortener's read-only endpoints and wraps
// them with the gzip middleware.
func NewHandler(cfg GzipConfig) http.Handler {
	links := make([]link, 200)
	for i := range links {
		links[i] = link{
			Code:   fmt.Sprintf("g%d", i+1),
			Target: fmt.Sprintf("https://example.com/articles/%d", i+1),
			Owner:  "u1",
		}
	}

	// Pseudo-random bytes stand in for a JPEG or PNG: already compressed
	// formats look like noise to gzip.
	avatar := make([]byte, 4096)
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range avatar {
		avatar[i] = byte(rng.UintN(256))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /links", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"items": links}); err != nil {
			http.Error(w, "encode failed", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("GET /avatar.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(avatar)
	})
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "pong")
	})
	return Gzip(cfg)(mux)
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
