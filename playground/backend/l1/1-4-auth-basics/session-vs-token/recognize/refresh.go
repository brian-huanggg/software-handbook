//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/refresh.go` to compile-check only.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var secret = []byte("load-from-env-in-production")

type Claims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
}

func sign(c Claims) string {
	b, _ := json.Marshal(c)
	p := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func verify(tok string) (Claims, error) {
	p, s, ok := strings.Cut(tok, ".")
	if !ok {
		return Claims{}, errors.New("malformed")
	}
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(p))
	got, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || !hmac.Equal(got, m.Sum(nil)) {
		return Claims{}, errors.New("bad signature")
	}
	b, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return Claims{}, err
	}
	var c Claims
	if err := json.Unmarshal(b, &c); err != nil {
		return Claims{}, err
	}
	if time.Now().Unix() >= c.Exp {
		return Claims{}, errors.New("expired")
	}
	return c, nil
}

func refresh(w http.ResponseWriter, r *http.Request) {
	ck, err := r.Cookie("refresh")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	c, err := verify(ck.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	access := sign(Claims{Sub: c.Sub, Role: c.Role, Exp: time.Now().Add(24 * time.Hour).Unix()})
	w.Write([]byte(access))
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "refresh", Value: "", MaxAge: -1, Path: "/"})
	w.WriteHeader(http.StatusNoContent)
}

func adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := verify(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil || c.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func main() {
	http.HandleFunc("POST /refresh", refresh)
	http.HandleFunc("POST /logout", logout)
	http.HandleFunc("DELETE /links/{code}", adminOnly(func(w http.ResponseWriter, r *http.Request) {}))
	http.ListenAndServe(":8080", nil)
}
