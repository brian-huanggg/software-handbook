//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/redirect.go` to compile-check only.
package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	http.HandleFunc("/", redirect)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// redirect asks the internal link service where a short code points,
// then sends the visitor there.
func redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	resp, err := http.Get("http://links.internal/resolve?code=" + url.QueryEscape(code))
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	target, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, string(target), http.StatusFound)
}
