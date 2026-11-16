//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/api.go` to compile-check only.
package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	mu    sync.Mutex
	links = map[string]string{}
)

func main() {
	http.Handle("PUT /links/{code}", requireToken(http.HandlerFunc(create)))
	http.HandleFunc("GET /{code}", redirect)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+os.Getenv("API_TOKEN") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
		next.ServeHTTP(w, r)
	})
}

func create(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	mu.Lock()
	links[r.PathValue("code")] = string(body)
	mu.Unlock()
}

func redirect(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	target, ok := links[r.PathValue("code")]
	mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
