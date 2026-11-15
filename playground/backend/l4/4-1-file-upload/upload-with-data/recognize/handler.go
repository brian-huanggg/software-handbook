//go:build recognize

// Package recognize is AI-generated code for the review exercise.
package recognize

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type Repo interface {
	Create(ctx context.Context, targetURL, key string) error
}

type Handler struct {
	Dir  string
	Repo Repo
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("preview")
	if err != nil {
		http.Error(w, "missing preview", http.StatusUnprocessableEntity)
		return
	}
	defer file.Close()

	target := r.FormValue("target_url")
	if _, err := url.ParseRequestURI(target); err != nil {
		http.Error(w, "bad target_url", http.StatusUnprocessableEntity)
		return
	}

	key := target[len(target)-6:] + ".img"
	if err := h.Repo.Create(r.Context(), target, key); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(filepath.Join(h.Dir, key))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	io.Copy(dst, file)
	dst.Close()
	w.WriteHeader(http.StatusCreated)
}
