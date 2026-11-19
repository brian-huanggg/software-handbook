package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func get(t *testing.T, h http.Handler, path, acceptEncoding string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCompressesJSON(t *testing.T) {
	h := NewHandler(DefaultGzip)
	plain := get(t, h, "/links", "")
	gz := get(t, h, "/links", "gzip")

	if got := gz.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := gz.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want Accept-Encoding", got)
	}
	if got := gz.Header().Get("Content-Length"); got != "" {
		t.Errorf("Content-Length = %q, want it removed", got)
	}

	zr, err := gzip.NewReader(gz.Body)
	if err != nil {
		t.Fatalf("not a gzip stream: %v", err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	if string(got) != plain.Body.String() {
		t.Error("decompressed body differs from the plain body")
	}
	if gz.Body.Len()*4 > plain.Body.Len() {
		t.Errorf("compressed %d of %d bytes, want under 25%%", gz.Body.Len(), plain.Body.Len())
	}
}

func TestVaryEvenWhenNotCompressed(t *testing.T) {
	rec := get(t, NewHandler(DefaultGzip), "/links", "")
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("compressed without Accept-Encoding")
	}
	if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q: a cache would reuse this plain body for gzip clients", got)
	}
}

func TestSkips(t *testing.T) {
	h := NewHandler(DefaultGzip)
	tests := []struct {
		name, path, accept string
	}{
		{"already compressed format", "/avatar.png", "gzip"},
		{"body below MinLength", "/ping", "gzip"},
		{"gzip refused with q=0", "/links", "gzip;q=0, identity"},
		{"identity only", "/links", "identity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, h, tt.path, tt.accept)
			if got := rec.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("Content-Encoding = %q, want none", got)
			}
		})
	}
}

func TestWildcardAccepts(t *testing.T) {
	rec := get(t, NewHandler(DefaultGzip), "/links", "*")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Accept-Encoding: * should allow gzip")
	}
}

func TestNoDoubleCompression(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "br")
		w.Write(make([]byte, 4096))
	})
	rec := get(t, Gzip(DefaultGzip)(inner), "/", "gzip")
	if got := rec.Header().Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding = %q, want the handler's br untouched", got)
	}
}

func TestBodylessStatuses(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	})
	rec := get(t, Gzip(DefaultGzip)(inner), "/", "gzip")
	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want 304", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "" || rec.Body.Len() != 0 {
		t.Error("304 must not carry a gzip header or body")
	}
}

func TestSniffsMissingContentType(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>" + string(make([]byte, 2000)) + "</html>"))
	})
	rec := get(t, Gzip(DefaultGzip)(inner), "/", "gzip")
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want it sniffed from the plain bytes", got)
	}
}

func TestTypeAllowed(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/html; charset=utf-8", true},
		{"image/svg+xml", true},
		{"image/png", false},
		{"video/mp4", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := typeAllowed(tt.ct, DefaultGzip.Types); got != tt.want {
			t.Errorf("typeAllowed(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}
