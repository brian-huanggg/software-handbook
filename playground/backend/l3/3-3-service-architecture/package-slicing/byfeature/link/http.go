package link

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Routes is the only HTTP entry point the rest of the program sees.
func (s *Service) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /links", s.create)
	mux.HandleFunc("GET /{code}", s.redirect)
}

func (s *Service) create(w http.ResponseWriter, r *http.Request) {
	var in struct{ Code, URL string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(s.Create(in.Code, in.URL)); err != nil {
		return // status already sent
	}
}

func (s *Service) redirect(w http.ResponseWriter, r *http.Request) {
	url, err := s.Resolve(r.PathValue("code"))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}
