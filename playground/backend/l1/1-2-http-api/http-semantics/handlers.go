package main

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

const maxBody = 2048

// Redirect handles GET /{code}. The mux also routes HEAD here.
func (s *Store) Redirect(w http.ResponseWriter, r *http.Request) {
	target, ok := s.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	// 302, not 301: browsers cache a 301 with no expiry, so a changed
	// target or a click counter would never see that visitor again.
	http.Redirect(w, r, target, http.StatusFound)
}

// Show handles GET /links/{code} and returns the target as plain text.
func (s *Store) Show(w http.ResponseWriter, r *http.Request) {
	target, ok := s.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// A write error means the client is gone; there is nobody left to tell.
	io.WriteString(w, target)
}

// PutLink handles PUT /links/{code}: create or replace, idempotent.
func (s *Store) PutLink(w http.ResponseWriter, r *http.Request) {
	target, ok := readTarget(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	if s.Put(code, target) {
		w.Header().Set("Location", "/links/"+code)
		w.WriteHeader(http.StatusCreated)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostLink handles POST /links: the server picks the code.
func (s *Store) PostLink(w http.ResponseWriter, r *http.Request) {
	target, ok := readTarget(w, r)
	if !ok {
		return
	}
	code := s.Add(target)
	w.Header().Set("Location", "/links/"+code)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, code)
}

// DeleteLink handles DELETE /links/{code}. A second DELETE gets 404, and
// that is still idempotent: the server state after both calls is the same.
func (s *Store) DeleteLink(w http.ResponseWriter, r *http.Request) {
	if !s.Delete(r.PathValue("code")) {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// readTarget validates the request body and writes the error response
// itself, so callers only need to return when ok is false.
func readTarget(w http.ResponseWriter, r *http.Request) (target string, ok bool) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusUnsupportedMediaType)
		return "", false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return "", false
		}
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return "", false
	}
	target = strings.TrimSpace(string(body))
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "target must be an absolute http(s) URL", http.StatusBadRequest)
		return "", false
	}
	return target, true
}
