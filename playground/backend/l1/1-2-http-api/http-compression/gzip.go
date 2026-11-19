package main

import (
	"compress/gzip"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// GzipConfig holds the three knobs every compression setup has.
type GzipConfig struct {
	// Level is a compress/gzip level, for example gzip.DefaultCompression.
	Level int
	// MinLength is the smallest body worth compressing, in bytes.
	MinLength int
	// Types lists compressible media types. "text/*" matches a whole family.
	Types []string
}

// DefaultGzip compresses text-like formats only.
var DefaultGzip = GzipConfig{
	Level:     gzip.DefaultCompression,
	MinLength: 1024,
	Types: []string{
		"text/*",
		"application/json",
		"application/javascript",
		"application/xml",
		"image/svg+xml",
	},
}

// Gzip returns a middleware that compresses responses when the client
// accepts gzip and the response is worth compressing.
func Gzip(cfg GzipConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The response now depends on Accept-Encoding, even when this
			// particular request is not compressed. Caches need to know.
			w.Header().Add("Vary", "Accept-Encoding")

			if !acceptsGzip(r.Header.Get("Accept-Encoding")) || r.Header.Get("Range") != "" {
				next.ServeHTTP(w, r)
				return
			}

			gw := &gzipWriter{ResponseWriter: w, cfg: cfg}
			defer gw.Close()
			next.ServeHTTP(gw, r)
		})
	}
}

// acceptsGzip reports whether an Accept-Encoding value allows gzip.
// "gzip;q=0" means "not acceptable", and "*" covers every coding not listed.
func acceptsGzip(header string) bool {
	star := false
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		name = strings.ToLower(strings.TrimSpace(name))

		q := 1.0
		if k, v, ok := strings.Cut(strings.TrimSpace(params), "="); ok && strings.TrimSpace(k) == "q" {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				q = f
			}
		}

		switch name {
		case "gzip", "x-gzip":
			return q > 0
		case "*":
			star = q > 0
		}
	}
	return star
}

// typeAllowed reports whether a Content-Type value matches the allowlist.
func typeAllowed(contentType string, types []string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	for _, t := range types {
		if fam, ok := strings.CutSuffix(t, "/*"); ok {
			if strings.HasPrefix(mt, fam+"/") {
				return true
			}
		} else if mt == t {
			return true
		}
	}
	return false
}

// gzipWriter buffers the first MinLength bytes so it can decide, after
// seeing how large the body really is, whether compressing is worth it.
type gzipWriter struct {
	http.ResponseWriter
	cfg     GzipConfig
	status  int
	buf     []byte
	decided bool
	gz      *gzip.Writer
}

func (w *gzipWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *gzipWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if !w.decided {
		w.buf = append(w.buf, p...)
		if len(w.buf) < w.cfg.MinLength {
			return len(p), nil
		}
		if err := w.decide(); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	if w.gz != nil {
		return w.gz.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

// decide fixes the headers and flushes the buffer. It runs once.
func (w *gzipWriter) decide() error {
	w.decided = true
	h := w.Header()

	// Without a Content-Type, net/http would sniff the gzip bytes and
	// guess wrong, so sniff the original bytes first.
	if h.Get("Content-Type") == "" && len(w.buf) > 0 {
		h.Set("Content-Type", http.DetectContentType(w.buf))
	}

	if w.shouldCompress() {
		h.Set("Content-Encoding", "gzip")
		// The old length describes the uncompressed body.
		h.Del("Content-Length")
		gz, err := gzip.NewWriterLevel(w.ResponseWriter, w.cfg.Level)
		if err != nil {
			return err
		}
		w.gz = gz
	}

	w.ResponseWriter.WriteHeader(w.status)
	buf := w.buf
	w.buf = nil
	if len(buf) == 0 {
		return nil
	}
	if w.gz != nil {
		_, err := w.gz.Write(buf)
		return err
	}
	_, err := w.ResponseWriter.Write(buf)
	return err
}

func (w *gzipWriter) shouldCompress() bool {
	h := w.Header()
	switch {
	case w.status < 200 || w.status == http.StatusNoContent ||
		w.status == http.StatusNotModified || w.status == http.StatusPartialContent:
		return false
	case h.Get("Content-Encoding") != "":
		return false // already compressed by the handler
	case strings.Contains(h.Get("Cache-Control"), "no-transform"):
		return false
	case len(w.buf) < w.cfg.MinLength:
		return false
	}
	ct := h.Get("Content-Type")
	// Event streams must reach the client chunk by chunk.
	if strings.HasPrefix(ct, "text/event-stream") {
		return false
	}
	return typeAllowed(ct, w.cfg.Types)
}

// Close finishes the response. A body shorter than MinLength is still
// sitting in the buffer and goes out uncompressed.
func (w *gzipWriter) Close() error {
	if !w.decided {
		if w.status == 0 {
			return nil // the handler wrote nothing; net/http sends 200
		}
		if err := w.decide(); err != nil {
			return err
		}
	}
	if w.gz != nil {
		return w.gz.Close()
	}
	return nil
}
