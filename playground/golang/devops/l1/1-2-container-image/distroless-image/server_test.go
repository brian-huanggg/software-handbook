package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCreateThenRedirect(t *testing.T) {
	h := newHandler()

	form := url.Values{"url": {"https://example.com/a/very/long/path"}}
	req := httptest.NewRequest("POST", "/links", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	code := strings.TrimSpace(rec.Body.String())

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/"+code, nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect status = %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://example.com/a/very/long/path" {
		t.Errorf("Location = %q", got)
	}
}

func TestRejectsNonHTTPURL(t *testing.T) {
	form := url.Values{"url": {"javascript:alert(1)"}}
	req := httptest.NewRequest("POST", "/links", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
