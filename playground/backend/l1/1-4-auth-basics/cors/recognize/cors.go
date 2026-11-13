//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/cors.go` to compile-check only.
package main

import (
	"log"
	"net/http"
	"strings"
)

// withCORS lets pages from sho.rt read the API with their cookies.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.HasSuffix(origin, "sho.rt") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/links", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`["todo"]`))
	})
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
