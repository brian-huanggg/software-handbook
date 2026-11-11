package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

const app = "http://127.0.0.1:9090"

func newTestAPI(t *testing.T) (http.Handler, *linkStore) {
	t.Helper()
	store := newLinkStore()
	api, err := newAPI(store, log.New(io.Discard, "", 0), []string{app})
	if err != nil {
		t.Fatal(err)
	}
	return api, store
}

func request(h http.Handler, method, path, origin string, cookie bool, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, nil)
	req.Header.Set("Origin", origin)
	if cookie {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionCookie})
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var preDelete = map[string]string{"Access-Control-Request-Method": "DELETE"}

// Without the CORS middleware in front, the mux answers a preflight with 405
// because only DELETE is registered for this path.
func TestMuxAloneRejectsPreflight(t *testing.T) {
	mux := newMux(newLinkStore(), log.New(io.Discard, "", 0))
	rec := request(mux, "OPTIONS", "/api/links/todo", app, false, preDelete)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestPreflightNeedsNoCookie(t *testing.T) {
	api, _ := newTestAPI(t)
	rec := request(api, "OPTIONS", "/api/links/todo", app, false, preDelete)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204 (not 401 from requireSession)", rec.Code)
	}
}

func TestAllowedOriginDeletes(t *testing.T) {
	api, store := newTestAPI(t)
	rec := request(api, "DELETE", "/api/links/todo", app, true, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if _, ok := store.links["todo"]; ok {
		t.Error("todo still exists")
	}
}

// A browser stops here: the refused preflight means it never sends DELETE.
func TestOtherOriginPreflightRefused(t *testing.T) {
	api, store := newTestAPI(t)
	rec := request(api, "OPTIONS", "/api/links/todo", "http://127.0.0.1:9191", false, preDelete)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if _, ok := store.links["todo"]; !ok {
		t.Error("todo was deleted by a preflight")
	}
}

func TestLoginSetsCookie(t *testing.T) {
	api, _ := newTestAPI(t)
	rec := request(api, "GET", "/login", "", false, nil)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != sessionCookie || !cookies[0].HttpOnly {
		t.Errorf("cookies = %v, want one HttpOnly session cookie", cookies)
	}
}
