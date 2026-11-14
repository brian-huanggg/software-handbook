package session_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"authkit/authn"
	"authkit/session"
)

type fixture struct {
	m     *session.Manager
	store session.Store
	now   time.Time
}

func newFixture(t *testing.T, opts session.Options) *fixture {
	t.Helper()
	f := &fixture{store: session.NewMemoryStore(), now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	opts.Now = func() time.Time { return f.now }
	if opts.TTL == 0 {
		opts.TTL = time.Hour
	}
	f.m = session.NewManager(f.store, opts)
	return f
}

func (f *fixture) login(t *testing.T, userID string, existing *http.Cookie) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	if existing != nil {
		r.AddCookie(existing)
	}
	if err := f.m.Login(w, r, userID); err != nil {
		t.Fatal(err)
	}
	cs := w.Result().Cookies()
	if len(cs) != 1 {
		t.Fatalf("want 1 cookie, got %d", len(cs))
	}
	return cs[0]
}

func reqWith(c *http.Cookie) *http.Request {
	r := httptest.NewRequest("GET", "/me", nil)
	if c != nil {
		r.AddCookie(c)
	}
	return r
}

func TestLogin_CookieAttributes(t *testing.T) {
	f := newFixture(t, session.Options{})
	c := f.login(t, "u1", nil)
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Fatalf("bad cookie flags: %+v", c)
	}
	if c.MaxAge != 3600 {
		t.Fatalf("MaxAge = %d, want 3600", c.MaxAge)
	}
	if len(c.Value) != 43 { // 32 bytes, unpadded base64url
		t.Fatalf("id length %d, want 43", len(c.Value))
	}
}

func TestLogin_InsecureCookiesForDev(t *testing.T) {
	f := newFixture(t, session.Options{InsecureCookies: true})
	if f.login(t, "u1", nil).Secure {
		t.Fatal("Secure should be off when InsecureCookies is set")
	}
}

func TestLogin_IDsAreUnique(t *testing.T) {
	f := newFixture(t, session.Options{})
	if f.login(t, "u1", nil).Value == f.login(t, "u1", nil).Value {
		t.Fatal("two logins produced the same id")
	}
}

func TestAuthenticate_RoundTripWithRoles(t *testing.T) {
	f := newFixture(t, session.Options{
		Roles: func(_ context.Context, id string) ([]string, error) { return []string{"admin"}, nil },
	})
	c := f.login(t, "u1", nil)
	got, err := f.m.Authenticate(reqWith(c))
	if err != nil {
		t.Fatal(err)
	}
	want := authn.Principal{UserID: "u1", Roles: []string{"admin"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAuthenticate_Rejects(t *testing.T) {
	f := newFixture(t, session.Options{})
	f.login(t, "u1", nil)
	for name, c := range map[string]*http.Cookie{
		"no cookie": nil,
		"empty":     {Name: "sid", Value: ""},
		"garbage":   {Name: "sid", Value: "!!!not-base64!!!"},
		"unknown":   {Name: "sid", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
	} {
		if _, err := f.m.Authenticate(reqWith(c)); !errors.Is(err, authn.ErrUnauthenticated) {
			t.Errorf("%s: got %v, want ErrUnauthenticated", name, err)
		}
	}
}

func TestAuthenticate_StoreHoldsHashNotRawID(t *testing.T) {
	f := newFixture(t, session.Options{})
	c := f.login(t, "u1", nil)
	if _, err := f.store.Get(context.Background(), []byte(c.Value)); !errors.Is(err, session.ErrNotFound) {
		t.Fatal("raw id must not be usable as a store key")
	}
}

func TestAuthenticate_ExpiredIsRejectedAndDeleted(t *testing.T) {
	f := newFixture(t, session.Options{TTL: time.Hour})
	c := f.login(t, "u1", nil)

	f.now = f.now.Add(time.Hour - time.Second)
	if _, err := f.m.Authenticate(reqWith(c)); err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	f.now = f.now.Add(2 * time.Second)
	if _, err := f.m.Authenticate(reqWith(c)); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("after expiry: got %v", err)
	}
	f.now = f.now.Add(-time.Hour) // even if the clock went back, row must be gone
	if _, err := f.m.Authenticate(reqWith(c)); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatal("expired session was not deleted")
	}
}

// Session fixation: Login must never adopt a client-supplied ID and must
// invalidate the pre-login session.
func TestLogin_RotatesID(t *testing.T) {
	f := newFixture(t, session.Options{})
	old := f.login(t, "anon", nil)

	fresh := f.login(t, "u1", old)
	if fresh.Value == old.Value {
		t.Fatal("session id not rotated on login")
	}
	if _, err := f.m.Authenticate(reqWith(old)); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatal("old session still valid after login")
	}
	if p, err := f.m.Authenticate(reqWith(fresh)); err != nil || p.UserID != "u1" {
		t.Fatalf("new session: %+v, %v", p, err)
	}
}

func TestLogin_IgnoresAttackerSuppliedID(t *testing.T) {
	f := newFixture(t, session.Options{})
	planted := &http.Cookie{Name: "sid", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
	fresh := f.login(t, "u1", planted)
	if fresh.Value == planted.Value {
		t.Fatal("adopted client-supplied session id")
	}
	if _, err := f.m.Authenticate(reqWith(planted)); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatal("planted id authenticates")
	}
}

func TestLogout(t *testing.T) {
	f := newFixture(t, session.Options{})
	c := f.login(t, "u1", nil)

	w := httptest.NewRecorder()
	if err := f.m.Logout(w, reqWith(c)); err != nil {
		t.Fatal(err)
	}
	cs := w.Result().Cookies()
	if len(cs) != 1 || cs[0].MaxAge >= 0 || cs[0].Value != "" {
		t.Fatalf("cookie not cleared: %+v", cs)
	}
	if _, err := f.m.Authenticate(reqWith(c)); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatal("session valid after logout")
	}
}

func TestLogout_WithoutSessionIsOK(t *testing.T) {
	f := newFixture(t, session.Options{})
	if err := f.m.Logout(httptest.NewRecorder(), reqWith(nil)); err != nil {
		t.Fatal(err)
	}
}

func TestManagerImplementsAuthenticator(t *testing.T) {
	var _ authn.Authenticator = (*session.Manager)(nil)
}
