//go:build recognize

// Package recognize is AI-generated code for the review exercise.
package recognize

import (
	"image"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct{ Dir string }

var allowed = map[string]bool{"image/png": true, "image/jpeg": true}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	file, hdr, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "missing avatar", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !allowed[hdr.Header.Get("Content-Type")] {
		http.Error(w, "only png or jpeg", http.StatusUnsupportedMediaType)
		return
	}
	if _, _, err := image.Decode(file); err != nil {
		http.Error(w, "not an image", http.StatusUnprocessableEntity)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	name := filepath.Clean(strings.TrimSpace(hdr.Filename))
	dst, err := os.Create(filepath.Join(h.Dir, name))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	http.ServeFile(w, r, filepath.Join(h.Dir, name))
}
