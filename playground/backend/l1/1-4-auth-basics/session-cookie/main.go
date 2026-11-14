// Command session-cookie serves a tiny login API for the short-URL service.
package main

import (
	"crypto/subtle"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"handbook/playground/backend/session-cookie/session"
)

//go:embed index.html
var indexHTML []byte

// demoVerify accepts one hard-coded account. A real service looks the user
// up and compares an argon2id hash instead.
func demoVerify(username, password string) (string, bool) {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte("alice")) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte("correct horse")) == 1
	if userOK && passOK {
		return "user-1", true
	}
	return "", false
}

func newMux(auth *session.Auth) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", auth.Login)
	mux.HandleFunc("POST /logout", auth.Logout)
	mux.Handle("GET /api/me", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := session.UserID(r.Context())
		fmt.Fprintf(w, "hello %s\n", id)
	})))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	return mux
}

func main() {
	store := session.NewStore(30 * time.Minute)
	go func() {
		for range time.Tick(time.Minute) {
			store.Sweep()
		}
	}()
	// Secure is false only because the demo runs on plain http://127.0.0.1.
	auth := session.NewAuth(store, demoVerify, session.Config{CookieName: "sid", Secure: false})
	srv := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           newMux(auth),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on http://%s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
