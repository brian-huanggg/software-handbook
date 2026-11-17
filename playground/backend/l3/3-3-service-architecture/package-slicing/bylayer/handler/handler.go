// Package handler holds every HTTP handler, layered style.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"handbook/playground/backend/package-slicing/bylayer/service"
)

type LinkHandler struct{ Svc *service.LinkService }

func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct{ Code, URL string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, h.Svc.Create(in.Code, in.URL))
}

func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	url, err := h.Svc.Resolve(r.PathValue("code"))
	if errors.Is(err, service.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

type StatsHandler struct{ Svc *service.StatsService }

func (h *StatsHandler) Top(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.Svc.Top(10))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line is already sent; all we can do is stop.
		return
	}
}
