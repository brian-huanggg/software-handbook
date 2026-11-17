package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"handbook/playground/backend/package-slicing/bylayer/service"
	"handbook/playground/backend/package-slicing/bylayer/store"
)

func TestRedirectCountsClick(t *testing.T) {
	clicks := store.NewClickStore()
	links := &LinkHandler{Svc: &service.LinkService{Links: store.NewLinkStore(), Clicks: clicks}}
	stats := &StatsHandler{Svc: &service.StatsService{Clicks: clicks}}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /links", links.Create)
	mux.HandleFunc("GET /stats/top", stats.Top)
	mux.HandleFunc("GET /{code}", links.Redirect)

	do(t, mux, "POST", "/links", `{"Code":"go","URL":"https://go.dev"}`, http.StatusCreated)
	do(t, mux, "GET", "/go", "", http.StatusFound)
	do(t, mux, "GET", "/nope", "", http.StatusNotFound)
	body := do(t, mux, "GET", "/stats/top", "", http.StatusOK)
	if !strings.Contains(body, `"Code":"go","Total":1`) {
		t.Fatalf("top = %s, want go with 1 click", body)
	}
}

func do(t *testing.T, h http.Handler, method, path, body string, want int) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	if rec.Code != want {
		t.Fatalf("%s %s = %d, want %d", method, path, rec.Code, want)
	}
	return rec.Body.String()
}
