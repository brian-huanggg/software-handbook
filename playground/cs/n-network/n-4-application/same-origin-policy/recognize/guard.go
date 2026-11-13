//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/guard.go` to compile-check only.
package main

import (
	"log"
	"net/http"
	"strings"
)

const trustedSite = "https://sho.rt"

var links = map[string]string{"todo": "https://example.com/todo"}

// guard only lets pages from our own site change links.
func guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if !strings.HasPrefix(origin, trustedSite) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/links/{key}", func(w http.ResponseWriter, r *http.Request) {
		delete(links, r.PathValue("key"))
		w.WriteHeader(http.StatusNoContent)
	})
	log.Fatal(http.ListenAndServe(":8080", guard(mux)))
}
