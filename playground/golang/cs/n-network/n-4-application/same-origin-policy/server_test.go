package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// The server answers a cross-origin request like any other. Hiding the
// response is the browser's job, so nothing here should fail.
func TestAPIServesCrossOriginRequest(t *testing.T) {
	var logs bytes.Buffer
	api := newAPI(newLinkStore(), log.New(&logs, "", 0))

	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	req.Header.Set("Origin", "http://127.0.0.1:9090")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := strings.TrimSpace(rec.Body.String()), `["go","mdn","todo"]`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want none in this chapter", got)
	}
	if !strings.Contains(logs.String(), `Origin="http://127.0.0.1:9090"`) {
		t.Errorf("log does not record the Origin header:\n%s", logs.String())
	}
}

// A simple cross-origin POST still changes data on the server, even though
// the calling page can never read the response.
func TestCrossOriginPostStillDeletes(t *testing.T) {
	store := newLinkStore()
	api := newAPI(store, log.New(&bytes.Buffer{}, "", 0))

	form := url.Values{"code": {"todo"}}
	req := httptest.NewRequest(http.MethodPost, "/api/links/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://127.0.0.1:9090")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if _, ok := store.links["todo"]; ok {
		t.Error(`link "todo" still exists after a cross-origin POST`)
	}
}

func TestPageEmbedsAPIOrigin(t *testing.T) {
	page := newPage("http://127.0.0.1:8080", log.New(&bytes.Buffer{}, "", 0))
	rec := httptest.NewRecorder()
	page.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	// html/template writes the Go string as a quoted JS string literal.
	if !strings.Contains(rec.Body.String(), `const api = "http://127.0.0.1:8080";`) {
		t.Errorf("page does not embed the API origin:\n%s", rec.Body.String())
	}
}
