package cors

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

const app = "https://app.sho.rt"

func newTestHandler(t *testing.T) (http.Handler, *bool) {
	t.Helper()
	mw, err := New(Policy{
		Origins:     []string{app},
		Methods:     []string{"GET", "DELETE"},
		Headers:     []string{"Content-Type"},
		Credentials: true,
		MaxAge:      10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	called := new(bool)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
	return mw(next), called
}

func do(h http.Handler, method, origin string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://api.sho.rt/api/links", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestNewRejectsOriginsBrowsersNeverSend(t *testing.T) {
	bad := []string{
		"*",
		"null",
		"https://app.sho.rt/", // trailing slash
		"https://APP.sho.rt",  // browsers lower-case the host
		"https://app.sho.rt:443",
		"app.sho.rt",
	}
	for _, o := range bad {
		if _, err := New(Policy{Origins: []string{o}}); !errors.Is(err, ErrBadOrigin) {
			t.Errorf("New(%q) err = %v, want ErrBadOrigin", o, err)
		}
	}
	for _, o := range []string{app, "http://127.0.0.1:9090"} {
		if _, err := New(Policy{Origins: []string{o}}); err != nil {
			t.Errorf("New(%q) = %v, want nil", o, err)
		}
	}
}

func TestAllowedOriginCanRead(t *testing.T) {
	h, called := newTestHandler(t)
	rec := do(h, "GET", app, nil)
	if !*called {
		t.Fatal("handler not called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != app {
		t.Errorf("Allow-Origin = %q, want %q", got, app)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, want true", got)
	}
	if !slices.Contains(rec.Header().Values("Vary"), "Origin") {
		t.Error("missing Vary: Origin")
	}
}

// The middleware hides the response from other origins but cannot stop the
// handler from running: CORS is not access control.
func TestOtherOriginStillReachesHandler(t *testing.T) {
	h, called := newTestHandler(t)
	rec := do(h, "GET", "https://evil.example", nil)
	if !*called {
		t.Error("handler not called; a simple request always arrives")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want none", got)
	}
	if !slices.Contains(rec.Header().Values("Vary"), "Origin") {
		t.Error("refusals must also send Vary: Origin")
	}
}

func TestPreflight(t *testing.T) {
	pre := map[string]string{"Access-Control-Request-Method": "DELETE"}
	t.Run("allowed", func(t *testing.T) {
		h, called := newTestHandler(t)
		rec := do(h, "OPTIONS", app, pre)
		if *called {
			t.Error("preflight reached the handler")
		}
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204", rec.Code)
		}
		want := map[string]string{
			"Access-Control-Allow-Origin":  app,
			"Access-Control-Allow-Methods": "GET, DELETE",
			"Access-Control-Allow-Headers": "Content-Type",
			"Access-Control-Max-Age":       "600",
		}
		for k, v := range want {
			if got := rec.Header().Get(k); got != v {
				t.Errorf("%s = %q, want %q", k, got, v)
			}
		}
	})
	t.Run("refused", func(t *testing.T) {
		h, called := newTestHandler(t)
		rec := do(h, "OPTIONS", "https://sho.rt.evil.example", pre)
		if *called {
			t.Error("refused preflight reached the handler")
		}
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, want none", got)
		}
	})
}
