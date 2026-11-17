package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "text/plain")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPutCreatesThenReplaces(t *testing.T) {
	h := NewHandler(NewStore())

	rec := do(h, "PUT", "/links/abc", "https://example.com/a")
	if rec.Code != http.StatusCreated {
		t.Fatalf("first PUT: got %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/links/abc" {
		t.Fatalf("Location = %q, want /links/abc", got)
	}
	if rec := do(h, "PUT", "/links/abc", "https://example.com/a"); rec.Code != http.StatusNoContent {
		t.Fatalf("second PUT: got %d, want 204", rec.Code)
	}
	if rec := do(h, "GET", "/links/abc", ""); rec.Body.String() != "https://example.com/a" {
		t.Fatalf("after two identical PUTs, target = %q", rec.Body.String())
	}
}

func TestPostCreatesNewLinkEachTime(t *testing.T) {
	h := NewHandler(NewStore())

	first := do(h, "POST", "/links", "https://example.com/a")
	second := do(h, "POST", "/links", "https://example.com/a")
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("POST codes = %d, %d, want 201, 201", first.Code, second.Code)
	}
	a, b := first.Header().Get("Location"), second.Header().Get("Location")
	if a == "" || a == b {
		t.Fatalf("Locations = %q, %q, want two different links", a, b)
	}
}

func TestDeleteTwice(t *testing.T) {
	h := NewHandler(NewStore())
	do(h, "PUT", "/links/abc", "https://example.com/a")

	if rec := do(h, "DELETE", "/links/abc", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("first DELETE: got %d, want 204", rec.Code)
	}
	if rec := do(h, "DELETE", "/links/abc", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("second DELETE: got %d, want 404", rec.Code)
	}
	if rec := do(h, "GET", "/abc", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("GET after DELETE: got %d, want 404", rec.Code)
	}
}

func TestRedirectAndHead(t *testing.T) {
	h := NewHandler(NewStore())
	do(h, "PUT", "/links/abc", "https://example.com/a")

	rec := do(h, "GET", "/abc", "")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://example.com/a" {
		t.Fatalf("GET: got %d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := do(h, "HEAD", "/abc", ""); rec.Code != http.StatusFound {
		t.Fatalf("HEAD: got %d, want 302", rec.Code)
	}
}

func TestUnknownMethodGets405WithAllow(t *testing.T) {
	h := NewHandler(NewStore())

	rec := do(h, "PATCH", "/links/abc", "https://example.com/a")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "DELETE, GET, HEAD, PUT" {
		t.Fatalf("Allow = %q", got)
	}
}

func TestPutRejectsBadInput(t *testing.T) {
	h := NewHandler(NewStore())

	tests := []struct {
		name        string
		contentType string
		body        string
		want        int
	}{
		{"wrong content type", "application/json", `{"url":"https://example.com"}`, http.StatusUnsupportedMediaType},
		{"not a URL", "text/plain", "hello", http.StatusBadRequest},
		{"javascript scheme", "text/plain", "javascript:alert(1)", http.StatusBadRequest},
		{"too large", "text/plain", "https://example.com/" + strings.Repeat("a", maxBody), http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/links/abc", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("got %d, want %d", rec.Code, tt.want)
			}
		})
	}
	if rec := do(h, "GET", "/links/abc", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("rejected PUTs must not create the link, got %d", rec.Code)
	}
}
