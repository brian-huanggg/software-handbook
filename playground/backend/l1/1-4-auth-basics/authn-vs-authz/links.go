package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type Link struct {
	Code    string `json:"code"`
	URL     string `json:"url"`
	OwnerID string `json:"owner_id"`
}

type Store struct {
	mu    sync.Mutex
	next  int
	links map[string]Link
}

func NewStore() *Store {
	return &Store{links: make(map[string]Link)}
}

func (s *Store) Create(ownerID, url string) Link {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	// A counter is fine for a demo; real codes must not be sequential.
	l := Link{Code: strconv.Itoa(s.next), URL: url, OwnerID: ownerID}
	s.links[l.Code] = l
	return l
}

func (s *Store) Get(code string) (Link, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.links[code]
	return l, ok
}

func (s *Store) Delete(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.links, code)
}

type Server struct {
	store *Store
	auth  *Authenticator
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /links", s.createLink)
	mux.HandleFunc("GET /links/{code}", s.getLink)
	mux.HandleFunc("DELETE /links/{code}", s.deleteLink)
	// Wrapping the whole mux means a route added later cannot forget authn.
	return s.auth.Middleware(mux)
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// The body has no owner field: the owner is whoever authn says is calling.
	var in struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.URL == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	l := s.store.Create(user.ID, in.URL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(l); err != nil {
		fmt.Println("encode response:", err)
	}
}

// load fetches the link and asks the policy, in that order, for every
// request that touches one specific link.
func (s *Server) load(w http.ResponseWriter, r *http.Request, a Action) (Link, bool) {
	user, ok := UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return Link{}, false
	}
	l, ok := s.store.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return Link{}, false
	}
	if !Allowed(user, a, l) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return Link{}, false
	}
	return l, true
}

func (s *Server) getLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.load(w, r, ActionView)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(l); err != nil {
		fmt.Println("encode response:", err)
	}
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.load(w, r, ActionDelete)
	if !ok {
		return
	}
	s.store.Delete(l.Code)
	w.WriteHeader(http.StatusNoContent)
}
