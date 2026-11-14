package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
)

const allowed = "http://localhost:3000"

func corsServe(t *testing.T, origins []string, method, origin string, hdr map[string]string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	called := false
	h := middleware.CORS(origins)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest(method, "/x", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, called
}

func TestCORS_AllowedOrigin(t *testing.T) {
	w, called := corsServe(t, []string{allowed}, "GET", allowed, nil)
	if !called || w.Code != 200 {
		t.Fatalf("called=%v code=%d", called, w.Code)
	}
	h := w.Header()
	if h.Get("Access-Control-Allow-Origin") != allowed {
		t.Fatalf("ACAO = %q", h.Get("Access-Control-Allow-Origin"))
	}
	if h.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentials not allowed")
	}
	if !strings.Contains(h.Get("Vary"), "Origin") {
		t.Fatal("Vary: Origin missing, shared caches could serve the wrong ACAO")
	}
}

func TestCORS_UnlistedOriginGetsNoHeaders(t *testing.T) {
	for _, origin := range []string{
		"http://evil.example",
		"http://localhost:3001",           // other port
		"https://localhost:3000",          // other scheme
		"http://localhost:3000.evil.test", // suffix trick
		"null",
	} {
		w, _ := corsServe(t, []string{allowed}, "GET", origin, nil)
		if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("origin %q got CORS headers: %v", origin, w.Header())
		}
	}
}

func TestCORS_NeverWildcard(t *testing.T) {
	w, _ := corsServe(t, []string{allowed}, "GET", allowed, nil)
	if w.Header().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("wildcard origin with credentials")
	}
}

func TestCORS_NoOriginHeaderIsUntouched(t *testing.T) {
	w, called := corsServe(t, []string{allowed}, "GET", "", nil)
	if !called || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("called=%v headers=%v", called, w.Header())
	}
	if !strings.Contains(w.Header().Get("Vary"), "Origin") {
		t.Fatal("Vary: Origin missing on no-Origin response; a cache could reuse it cross-origin")
	}
}

func TestCORS_Preflight(t *testing.T) {
	pre := map[string]string{
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "content-type, x-requested-with",
	}
	w, called := corsServe(t, []string{allowed}, "OPTIONS", allowed, pre)
	if called {
		t.Fatal("preflight must not reach the handler")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("code %d, want 204", w.Code)
	}
	h := w.Header()
	if h.Get("Access-Control-Allow-Origin") != allowed || h.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("headers: %v", h)
	}
	if !strings.Contains(h.Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("methods: %q", h.Get("Access-Control-Allow-Methods"))
	}
	for _, want := range []string{"Content-Type", "Authorization", "X-Requested-With"} {
		if !strings.Contains(h.Get("Access-Control-Allow-Headers"), want) {
			t.Fatalf("allow-headers %q lacks %s", h.Get("Access-Control-Allow-Headers"), want)
		}
	}
	if h.Get("Access-Control-Max-Age") == "" {
		t.Fatal("no Max-Age")
	}
}

func TestCORS_PreflightFromUnlistedOriginIsRefused(t *testing.T) {
	pre := map[string]string{"Access-Control-Request-Method": "POST"}
	w, called := corsServe(t, []string{allowed}, "OPTIONS", "http://evil.example", pre)
	if called || w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("called=%v code=%d headers=%v", called, w.Code, w.Header())
	}
}

func TestCORS_EmptyAllowListIsPassThrough(t *testing.T) {
	w, called := corsServe(t, nil, "GET", allowed, nil)
	if !called || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("called=%v headers=%v", called, w.Header())
	}
}

func TestCORS_PanicsOnBadOrigins(t *testing.T) {
	for _, bad := range []string{"*", "", "localhost:3000", "http://localhost:3000/", "http://localhost:3000/app", "ftp://x"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("CORS(%q) should panic", bad)
				}
			}()
			middleware.CORS([]string{bad})
		}()
	}
}

func TestValidateOrigin(t *testing.T) {
	for _, ok := range []string{"http://localhost:3000", "https://app.example.com"} {
		if err := middleware.ValidateOrigin(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"*", "", "app.example.com", "http://x/", "http://x/y", "http://", "javascript:alert(1)"} {
		if middleware.ValidateOrigin(bad) == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
