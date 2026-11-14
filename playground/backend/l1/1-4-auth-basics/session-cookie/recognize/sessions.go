//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/sessions.go` to compile-check only.
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var (
	mu       sync.Mutex
	sessions = map[string]string{} // session ID -> user ID
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newSessionID() string {
	return fmt.Sprintf("%016x%016x", rand.Uint64(), rand.Uint64())
}

func login(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticate(r.FormValue("username"), r.FormValue("password"))
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	id := newSessionID()
	mu.Lock()
	sessions[id] = userID
	mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:   "sid",
		Value:  id,
		Path:   "/",
		MaxAge: 86400,
	})
	w.WriteHeader(http.StatusNoContent)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "sid",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func currentUser(r *http.Request) (string, bool) {
	c, err := r.Cookie("sid")
	if err != nil {
		return "", false
	}
	mu.Lock()
	defer mu.Unlock()
	userID, ok := sessions[c.Value]
	return userID, ok
}

func authenticate(username, password string) (string, bool) {
	return "user-1", username == "alice" && password == "correct horse"
}

func main() {
	http.HandleFunc("POST /login", login)
	http.HandleFunc("POST /logout", logout)
	http.ListenAndServe("127.0.0.1:8080", nil)
}
