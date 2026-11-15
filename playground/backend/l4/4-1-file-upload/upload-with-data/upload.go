package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
)

// Handler accepts POST /links as multipart/form-data:
//
//	target_url, title  (text parts, must come first)
//	preview            (file part, last)
type Handler struct {
	Store    DirStore
	Repo     Repo
	MaxBody  int64 // whole request body
	MaxFile  int64 // the preview file alone
	MaxField int64 // one text field
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Cap the whole body first: every later read, even a field, is bounded.
	r.Body = http.MaxBytesReader(w, r.Body, h.MaxBody)

	mr, err := r.MultipartReader() // streams parts, buffers no file
	if err != nil {
		http.Error(w, "expected multipart/form-data", http.StatusBadRequest)
		return
	}

	fields := map[string]string{}
	var tmp string
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			h.fail(w, err, tmp)
			return
		}
		name := part.FormName()

		if tmp != "" {
			// Anything after the file would be read only after it is on disk.
			h.fail(w, errFieldAfterFile, tmp)
			return
		}
		if name != "preview" {
			b, err := io.ReadAll(io.LimitReader(part, h.MaxField+1))
			if err != nil {
				h.fail(w, err, tmp)
				return
			}
			if int64(len(b)) > h.MaxField {
				h.fail(w, errFieldTooLarge, tmp)
				return
			}
			fields[name] = string(b)
			continue
		}

		// The file arrives: validate every field we already have *before*
		// reading a single byte of the file.
		if err := validate(fields); err != nil {
			h.fail(w, err, tmp)
			return
		}
		tmp, err = h.Store.Stage(part, h.MaxFile)
		if err != nil {
			h.fail(w, err, "")
			return
		}
	}

	if tmp == "" {
		h.fail(w, errNoFile, "")
		return
	}
	h.commit(w, r, fields, tmp)
}

func (h *Handler) commit(w http.ResponseWriter, r *http.Request, fields map[string]string, tmp string) {
	key, err := newKey()
	if err != nil {
		h.fail(w, err, tmp)
		return
	}
	// Order matters: file first, record second. A failure in between leaves
	// an unreferenced file (harmless, cleaned later), never a record that
	// points at a file that does not exist.
	if err := h.Store.Commit(tmp, key); err != nil {
		h.fail(w, err, tmp)
		return
	}
	l := Link{Code: key[:7], TargetURL: fields["target_url"], Title: fields["title"], PreviewKey: key}
	if err := h.Repo.Create(r.Context(), l); err != nil {
		if rmErr := h.Store.Remove(key); rmErr != nil {
			log.Printf("orphan object %s: %v", key, rmErr) // left for the sweeper
		}
		h.fail(w, err, "")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

var (
	errFieldAfterFile = errors.New("text fields must come before the file")
	errFieldTooLarge  = errors.New("field too large")
	errNoFile         = errors.New("missing preview file")
	errBadURL         = errors.New("target_url must be an absolute http(s) URL")
)

func validate(fields map[string]string) error {
	u, err := url.ParseRequestURI(fields["target_url"])
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errBadURL
	}
	return nil
}

// fail maps an error to a status code and removes the staged file, if any.
func (h *Handler) fail(w http.ResponseWriter, err error, tmp string) {
	if tmp != "" {
		h.Store.Discard(tmp)
	}
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, errTooLarge), errors.Is(err, errFieldTooLarge):
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
	case errors.Is(err, errFieldAfterFile):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errBadURL), errors.Is(err, errNoFile):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		log.Printf("upload: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
