package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() (*Server, http.Handler) {
	srv := &Server{
		store: NewStore(),
		auth: &Authenticator{users: map[string]User{
			"alice-token": {ID: "alice", Role: RoleUser},
			"bob-token":   {ID: "bob", Role: RoleUser},
			"root-token":  {ID: "root", Role: RoleAdmin},
		}},
	}
	return srv, srv.Routes()
}

func do(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAccessMatrix(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		token  string
		want   int
	}{
		{"no token", "GET", "/links/1", "", http.StatusUnauthorized},
		{"unknown token", "GET", "/links/1", "nope", http.StatusUnauthorized},
		{"owner views", "GET", "/links/1", "alice-token", http.StatusOK},
		{"other user views", "GET", "/links/1", "bob-token", http.StatusForbidden},
		{"admin views", "GET", "/links/1", "root-token", http.StatusForbidden},
		{"other user deletes", "DELETE", "/links/1", "bob-token", http.StatusForbidden},
		{"missing link", "GET", "/links/99", "alice-token", http.StatusNotFound},
		{"admin deletes", "DELETE", "/links/1", "root-token", http.StatusNoContent},
		{"owner deletes", "DELETE", "/links/1", "alice-token", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, h := newTestServer()
			srv.store.Create("alice", "https://example.com/a")
			if got := do(h, tt.method, tt.path, tt.token, "").Code; got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestForbiddenDeleteKeepsLink(t *testing.T) {
	srv, h := newTestServer()
	srv.store.Create("alice", "https://example.com/a")
	do(h, "DELETE", "/links/1", "bob-token", "")
	if _, ok := srv.store.Get("1"); !ok {
		t.Fatal("link was deleted by a user who does not own it")
	}
}

func TestOwnerComesFromIdentityNotBody(t *testing.T) {
	srv, h := newTestServer()
	rec := do(h, "POST", "/links", "bob-token", `{"url":"https://example.com","owner_id":"alice"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	l, ok := srv.store.Get("1")
	if !ok || l.OwnerID != "bob" {
		t.Fatalf("owner = %q, want bob", l.OwnerID)
	}
}
