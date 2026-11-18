package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type linkInput struct {
	Target string `json:"target"`
	Owner  string `json:"owner"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

// readInput decodes the body and checks the target. Full input validation
// is the topic of the next article; this is the minimum the routes need.
func readInput(w http.ResponseWriter, r *http.Request) (linkInput, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var in linkInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return in, false
		}
		http.Error(w, "body must be a JSON object", http.StatusBadRequest)
		return in, false
	}
	u, err := url.Parse(in.Target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "target must be an http or https URL", http.StatusBadRequest)
		return in, false
	}
	return in, true
}

// CreateLink handles POST /links. The server picks the code, so the client
// learns where the new resource lives from the Location header.
func (s *Store) CreateLink(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	l := s.Add(in.Target, in.Owner)
	w.Header().Set("Location", "/links/"+l.Code)
	writeJSON(w, http.StatusCreated, l)
}

// PutLink handles PUT /links/{code}. The client picks the code and the body
// replaces the whole resource, so sending it twice leaves the same state.
func (s *Store) PutLink(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	l, created := s.Put(r.PathValue("code"), in.Target, in.Owner)
	if created {
		w.Header().Set("Location", "/links/"+l.Code)
		writeJSON(w, http.StatusCreated, l)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Store) ShowLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Store) DeleteLink(w http.ResponseWriter, r *http.Request) {
	if !s.Delete(r.PathValue("code")) {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DisableLink handles POST /links/{code}/disable: an action that is not
// plain create, read, replace or delete gets its own sub-path.
func (s *Store) DisableLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.Disable(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// Redirect handles GET /{code}, the public short URL.
func (s *Store) Redirect(w http.ResponseWriter, r *http.Request) {
	l, ok := s.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if l.Disabled {
		http.Error(w, "link disabled", http.StatusGone)
		return
	}
	http.Redirect(w, r, l.Target, http.StatusFound)
}

func (s *Store) ListLinks(w http.ResponseWriter, r *http.Request) {
	s.list(w, r, "")
}

// ListOwnerLinks handles GET /users/{owner}/links: the owner is part of the
// path because the links belong to that user.
func (s *Store) ListOwnerLinks(w http.ResponseWriter, r *http.Request) {
	s.list(w, r, r.PathValue("owner"))
}

// list serves both collection URLs. Filtering and paging come from the
// query string because they narrow a collection without naming a new one.
func (s *Store) list(w http.ResponseWriter, r *http.Request, owner string) {
	q := r.URL.Query()
	limit, err := intParam(q, "limit", 20, 1, 100)
	if err == nil {
		var offset int
		offset, err = intParam(q, "offset", 0, 0, 1<<30)
		if err == nil {
			host := q.Get("host")
			items := s.List(func(l Link) bool {
				if owner != "" && l.Owner != owner {
					return false
				}
				if host == "" {
					return true
				}
				u, perr := url.Parse(l.Target)
				return perr == nil && u.Hostname() == host
			}, limit, offset)
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
			return
		}
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func intParam(q url.Values, name string, def, lo, hi int) (int, error) {
	raw := q.Get(name)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, lo, hi)
	}
	return n, nil
}
