package session

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testAuth(secure bool) (*Auth, *Store, http.Handler) {
	store := NewStore(time.Hour)
	verify := func(u, p string) (string, bool) { return "user-1", u == "alice" && p == "pw" }
	auth := NewAuth(store, verify, Config{CookieName: "sid", Secure: secure})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", auth.Login)
	mux.HandleFunc("POST /logout", auth.Logout)
	mux.Handle("GET /me", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := UserID(r.Context())
		_, _ = w.Write([]byte(id))
	})))
	return auth, store, mux
}

func do(h http.Handler, method, path, body, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "sid", Value: cookie})
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func loginForm(user, pass string) string {
	return url.Values{"username": {user}, "password": {pass}}.Encode()
}

func sidCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "sid" {
			return c
		}
	}
	t.Fatal("no sid cookie in response")
	return nil
}

func TestLoginSetsHardenedCookie(t *testing.T) {
	_, _, h := testAuth(true)
	c := sidCookie(t, do(h, "POST", "/login", loginForm("alice", "pw"), ""))
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Fatalf("cookie attributes = %+v", c)
	}
	if c.Domain != "" {
		t.Fatalf("Domain = %q, want empty so subdomains cannot read it", c.Domain)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	_, _, h := testAuth(true)
	rec := do(h, "POST", "/login", loginForm("alice", "nope"), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("failed login must not set a cookie")
	}
}

func TestRequire(t *testing.T) {
	_, _, h := testAuth(true)
	if rec := do(h, "GET", "/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", rec.Code)
	}
	if rec := do(h, "GET", "/me", "", "forged-id"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("forged cookie: status = %d, want 401", rec.Code)
	}
	id := sidCookie(t, do(h, "POST", "/login", loginForm("alice", "pw"), "")).Value
	rec := do(h, "GET", "/me", "", id)
	if rec.Code != http.StatusOK || rec.Body.String() != "user-1" {
		t.Fatalf("valid cookie: %d %q", rec.Code, rec.Body.String())
	}
}

func TestLogoutKillsCopiedCookie(t *testing.T) {
	_, _, h := testAuth(true)
	id := sidCookie(t, do(h, "POST", "/login", loginForm("alice", "pw"), "")).Value
	do(h, "POST", "/logout", "", id)
	// Someone copied the cookie before logout and replays it afterwards.
	if rec := do(h, "GET", "/me", "", id); rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed cookie after logout: status = %d, want 401", rec.Code)
	}
}

func TestLoginRotatesID(t *testing.T) {
	_, store, h := testAuth(true)
	planted, err := store.Create("nobody")
	if err != nil {
		t.Fatal(err)
	}
	fresh := sidCookie(t, do(h, "POST", "/login", loginForm("alice", "pw"), planted.ID)).Value
	if fresh == planted.ID {
		t.Fatal("login reused the ID the client supplied")
	}
	if _, err := store.Get(planted.ID); err == nil {
		t.Fatal("old ID still valid after login")
	}
}
