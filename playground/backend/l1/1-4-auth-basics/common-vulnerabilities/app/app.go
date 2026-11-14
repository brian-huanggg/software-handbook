// Package app is a tiny short-URL admin API that exists in two versions:
// Vulnerable, with the three classic mistakes, and Safe, with the fixes.
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// sessionID is a fixed value so the demo needs no login flow.
const sessionID = "alice-session"

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

type Server struct {
	mu    sync.Mutex
	links map[string]string
	user  User
	log   *slog.Logger
}

func NewServer(log *slog.Logger) *Server {
	return &Server{
		links: map[string]string{"go": "https://go.dev"},
		user: User{
			ID:           1,
			Email:        "alice@example.com",
			PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$aGFzaGhhc2hoYXNo",
		},
		log: log,
	}
}

func (s *Server) HasLink(slug string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.links[slug]
	return ok
}

func (s *Server) authed(r *http.Request) bool {
	c, err := r.Cookie("sid")
	return err == nil && c.Value == sessionID
}

// Vulnerable wires the handlers that fall for XSS, CSRF and data exposure.
func (s *Server) Vulnerable() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /report", reportUnsafe)
	mux.HandleFunc("POST /links/delete", s.deleteLink)
	mux.HandleFunc("GET /me", s.meUnsafe)
	mux.HandleFunc("GET /links/lookup", s.lookupUnsafe)
	return mux
}

// Safe wires the same routes with the fixes applied.
func (s *Server) Safe() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /report", reportSafe)
	mux.HandleFunc("POST /links/delete", s.deleteLink)
	mux.HandleFunc("GET /me", s.meSafe)
	mux.HandleFunc("GET /links/lookup", s.lookupSafe)
	// Rejects cross-site POST/PUT/DELETE; GET is never checked.
	return http.NewCrossOriginProtection().Handler(mux)
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	delete(s.links, r.FormValue("slug"))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// lookup stands in for a database call that fails the way real drivers do:
// the error text carries the query and the internal address.
func lookup(slug string) error {
	return fmt.Errorf("query %q: dial tcp 10.0.3.7:5432: connection refused",
		"SELECT target FROM links WHERE slug = '"+slug+"'")
}

func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Error("encode response", "err", err)
	}
}

var errLookup = errors.New("lookup failed")
