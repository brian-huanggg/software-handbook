package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHandler(t *testing.T) (http.Handler, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	return NewHandler(NewStore(), "secret", log.New(&logs, "", 0)), &logs
}

func do(h http.Handler, method, path, body, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateThenRedirect(t *testing.T) {
	h, _ := newTestHandler(t)

	if rec := do(h, "PUT", "/links/abc", "https://example.com/long", "Bearer secret"); rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201", rec.Code)
	}
	rec := do(h, "GET", "/abc", "", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect: got %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://example.com/long" {
		t.Fatalf("Location = %q", got)
	}
}

func TestUnknownCodeIs404(t *testing.T) {
	h, _ := newTestHandler(t)
	if rec := do(h, "GET", "/nope", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
}

func TestMuxMethodMatching(t *testing.T) {
	h, _ := newTestHandler(t)
	rec := do(h, "POST", "/abc", "", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow = %q, want it to contain GET", allow)
	}
}

func TestCreateRequiresToken(t *testing.T) {
	h, _ := newTestHandler(t)
	for _, auth := range []string{"", "Bearer wrong"} {
		if rec := do(h, "PUT", "/links/abc", "https://example.com", auth); rec.Code != http.StatusUnauthorized {
			t.Errorf("auth %q: got %d, want 401", auth, rec.Code)
		}
	}
}

func TestLoggingRecordsStatus(t *testing.T) {
	h, logs := newTestHandler(t)
	do(h, "GET", "/nope", "", "")
	if !strings.Contains(logs.String(), "GET /nope -> 404") {
		t.Fatalf("log = %q", logs.String())
	}
}

func TestRecoverTurnsPanicInto500(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := Chain(boom, Logging(logger), Recover(logger))

	rec := do(h, "GET", "/", "", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", rec.Code)
	}
	if !strings.Contains(logs.String(), "-> 500") {
		t.Fatalf("panic was not logged as 500: %q", logs.String())
	}
}

func TestChainOrder(t *testing.T) {
	var order []string
	mk := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+" in")
				next.ServeHTTP(w, r)
				order = append(order, name+" out")
			})
		}
	}
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	}), mk("A"), mk("B"))
	do(h, "GET", "/", "", "")

	want := "A in,B in,handler,B out,A out"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestServerHasTimeouts(t *testing.T) {
	srv := NewServer(":0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout == 0 || srv.IdleTimeout == 0 {
		t.Fatal("server must set ReadHeaderTimeout and IdleTimeout")
	}
}

func TestOverLimitBodyRejected(t *testing.T) {
	h, _ := newTestHandler(t)
	big := strings.Repeat("x", 5000)
	rec := do(h, "PUT", "/links/abc", big, "Bearer secret")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", rec.Code)
	}
}
