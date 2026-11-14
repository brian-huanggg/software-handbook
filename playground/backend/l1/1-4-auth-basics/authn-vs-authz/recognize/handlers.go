//go:build ignore

// AI-generated. Not part of the example's tests; it only compiles when
// copied next to the example's other files.
package main

import (
	"encoding/json"
	"net/http"
)

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL     string `json:"url"`
		OwnerID string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	l := s.store.Create(in.OwnerID, in.URL)
	json.NewEncoder(w).Encode(l)
}

func (s *Server) getLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.store.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(l)
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFrom(r.Context())
	l, ok := s.store.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	isAdmin := r.Header.Get("X-Role") == "admin"
	if l.OwnerID != user.ID && !isAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.store.Delete(l.Code)
	w.WriteHeader(http.StatusNoContent)
}
