package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

type requestIDKey struct{}

const requestIDHeader = "X-Request-Id"

// RequestIDFrom returns the request ID stored by RequestID, or "".
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID gives every request an ID: the caller's X-Request-Id when it is
// short and printable ASCII (so it cannot inject log lines), otherwise a
// fresh random one. It is stored in the context and echoed in the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range []byte(id) {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // never fails on supported platforms
	return hex.EncodeToString(b)
}

// WithRequestID wraps h so records logged with a request context carry
// request_id.
func WithRequestID(h slog.Handler) slog.Handler { return requestIDHandler{h} }

type requestIDHandler struct{ slog.Handler }

func (h requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h requestIDHandler) WithAttrs(a []slog.Attr) slog.Handler {
	return requestIDHandler{h.Handler.WithAttrs(a)}
}

func (h requestIDHandler) WithGroup(n string) slog.Handler {
	return requestIDHandler{h.Handler.WithGroup(n)}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// AccessLog logs one line per request: method, path, status, size, duration
// and client IP. Paths in skip (probes) are not logged. Only r.URL.Path is
// logged, never the query string, which could carry secrets.
func AccessLog(next http.Handler, skip ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slices.Contains(skip, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		slog.LogAttrs(r.Context(), slog.LevelInfo, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sw.status),
			slog.Int("bytes", sw.bytes),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", ClientIP(r)),
		)
	})
}
