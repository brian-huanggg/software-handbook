//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/origins.go` to compile-check only.
package main

import (
	"log"
	"net/http"
	"strings"
)

// allowedOrigins lists the front ends that may read the API with cookies.
var allowedOrigins = map[string]bool{
	"https://app.sho.rt":    true,
	"http://app.sho.rt":     true,
	"http://localhost:3000": true,
	"null":                  true,
}

// originAllowed accepts listed front ends and any sho.rt subdomain.
func originAllowed(origin string) bool {
	if allowedOrigins[origin] {
		return true
	}
	return strings.HasSuffix(origin, "sho.rt")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		if origin := r.Header.Get("Origin"); originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"user":"alice"}`))
	})
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
