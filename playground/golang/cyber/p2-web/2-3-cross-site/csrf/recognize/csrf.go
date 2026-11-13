//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/csrf.go` to compile-check only.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
)

// csrfGuard blocks cross-site form posts to state-changing endpoints.
func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if origin != "" && !strings.HasPrefix(origin, "http://victim.test") {
				http.Error(w, "cross-site request blocked", http.StatusForbidden)
				return
			}
			c, err := r.Cookie("csrf")
			if err != nil || c.Value != r.FormValue("csrf_token") {
				http.Error(w, "bad token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// issueToken hands out a random token in a cookie; the form echoes it back.
func issueToken(w http.ResponseWriter) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	tok := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{Name: "csrf", Value: tok, Path: "/"})
	return tok
}

func setTarget(w http.ResponseWriter, r *http.Request) {
	log.Printf("alice's /a1 now points to %s", r.FormValue("url"))
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/settings/target", setTarget) // any method
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", csrfGuard(mux)))
}
