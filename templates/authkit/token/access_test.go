package token_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
	"github.com/brian-huanggg/software-handbook/templates/authkit/token"
)

var secret = []byte("0123456789abcdef0123456789abcdef") // 32 bytes

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newAccess(t *testing.T, c *clock) *token.Access {
	t.Helper()
	a, err := token.NewAccess(token.AccessOptions{Secret: secret, Issuer: "authkit", Now: c.now})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newClock() *clock { return &clock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)} }

func TestNewAccess_RejectsShortSecret(t *testing.T) {
	if _, err := token.NewAccess(token.AccessOptions{Secret: []byte("short"), Issuer: "x"}); err == nil {
		t.Fatal("expected error for short secret")
	}
	if _, err := token.NewAccess(token.AccessOptions{Secret: secret}); err == nil {
		t.Fatal("expected error for empty issuer")
	}
}

func TestAccess_RoundTrip(t *testing.T) {
	c := newClock()
	a := newAccess(t, c)
	want := authn.Principal{UserID: "u1", Roles: []string{"admin", "user"}}

	tok, exp, err := a.Issue(want)
	if err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(c.t.Add(15 * time.Minute)) {
		t.Fatalf("exp = %v, want now+15m", exp)
	}
	got, err := a.Verify(tok)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Verify = %+v, %v; want %+v", got, err, want)
	}
}

func TestAccess_Expiry(t *testing.T) {
	c := newClock()
	a := newAccess(t, c)
	tok, _, _ := a.Issue(authn.Principal{UserID: "u1"})

	c.t = c.t.Add(15*time.Minute - time.Second)
	if _, err := a.Verify(tok); err != nil {
		t.Fatalf("just before expiry: %v", err)
	}
	c.t = c.t.Add(2 * time.Second)
	if _, err := a.Verify(tok); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("after expiry: got %v", err)
	}
}

// sign builds a token by hand. It carries kid, the id of the test secret, so
// each forged case fails for its own reason rather than for an unknown kid.
func sign(t *testing.T, kid string, m jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(m, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAccess_RejectsForgedOrInvalid(t *testing.T) {
	c := newClock()
	a := newAccess(t, c)
	good := func() jwt.MapClaims {
		return jwt.MapClaims{
			"sub": "u1", "iss": "authkit",
			"iat": c.t.Unix(), "exp": c.t.Add(time.Minute).Unix(),
		}
	}
	with := func(k string, v any) jwt.MapClaims { m := good(); m[k] = v; return m }
	without := func(k string) jwt.MapClaims { m := good(); delete(m, k); return m }

	valid, _, _ := a.Issue(authn.Principal{UserID: "u1"})
	parts := strings.Split(valid, ".")
	parsed, _, _ := jwt.NewParser().ParseUnverified(valid, jwt.MapClaims{})
	kid := parsed.Header["kid"].(string)

	cases := map[string]string{
		"alg=none":        sign(t, kid, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, good()),
		"alg=HS512":       sign(t, kid, jwt.SigningMethodHS512, secret, good()),
		"wrong secret":    sign(t, kid, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), good()),
		"wrong issuer":    sign(t, kid, jwt.SigningMethodHS256, secret, with("iss", "evil")),
		"missing issuer":  sign(t, kid, jwt.SigningMethodHS256, secret, without("iss")),
		"missing exp":     sign(t, kid, jwt.SigningMethodHS256, secret, without("exp")),
		"expired":         sign(t, kid, jwt.SigningMethodHS256, secret, with("exp", c.t.Add(-time.Minute).Unix())),
		"iat in future":   sign(t, kid, jwt.SigningMethodHS256, secret, with("iat", c.t.Add(time.Hour).Unix())),
		"missing sub":     sign(t, kid, jwt.SigningMethodHS256, secret, without("sub")),
		"empty sub":       sign(t, kid, jwt.SigningMethodHS256, secret, with("sub", "")),
		"tampered sig":    parts[0] + "." + parts[1] + "." + strings.Repeat("A", len(parts[2])),
		"tampered claims": parts[0] + "." + "eyJzdWIiOiJhZG1pbiJ9" + "." + parts[2],
		"unknown kid":     sign(t, "nope", jwt.SigningMethodHS256, secret, good()),
		"no kid":          sign(t, "", jwt.SigningMethodHS256, secret, good()),
		"garbage":         "not-a-jwt",
		"empty":           "",
	}
	for name, tok := range cases {
		if _, err := a.Verify(tok); !errors.Is(err, authn.ErrUnauthenticated) {
			t.Errorf("%s: got %v, want ErrUnauthenticated", name, err)
		}
	}
}

func TestAccess_KeyRotation(t *testing.T) {
	c := newClock()
	other := []byte("fedcba9876543210fedcba9876543210")
	old := newAccess(t, c)
	oldTok, _, _ := old.Issue(authn.Principal{UserID: "u1"})

	rotated, err := token.NewAccess(token.AccessOptions{Secret: other, PreviousSecrets: [][]byte{secret}, Issuer: "authkit", Now: c.now})
	if err != nil {
		t.Fatal(err)
	}
	if p, err := rotated.Verify(oldTok); err != nil || p.UserID != "u1" {
		t.Fatalf("token from previous key: %v %v", p, err)
	}
	newTok, _, _ := rotated.Issue(authn.Principal{UserID: "u2"})
	if _, err := old.Verify(newTok); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("old instance accepted a token from a key it does not know: %v", err)
	}

	// Once the previous key is dropped, its tokens stop verifying.
	dropped, _ := token.NewAccess(token.AccessOptions{Secret: other, Issuer: "authkit", Now: c.now})
	if _, err := dropped.Verify(oldTok); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("token from dropped key accepted: %v", err)
	}
}

func TestAccess_IssuedTokenHasKid(t *testing.T) {
	tok, _, _ := newAccess(t, newClock()).Issue(authn.Principal{UserID: "u1"})
	parsed, _, err := jwt.NewParser().ParseUnverified(tok, jwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if kid, _ := parsed.Header["kid"].(string); kid == "" {
		t.Fatalf("no kid in header %v", parsed.Header)
	}
}

func TestNewAccess_RejectsBadPreviousSecrets(t *testing.T) {
	for name, prev := range map[string][][]byte{
		"short":     {[]byte("short")},
		"duplicate": {secret},
	} {
		if _, err := token.NewAccess(token.AccessOptions{Secret: secret, PreviousSecrets: prev, Issuer: "x"}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestBearer_Authenticate(t *testing.T) {
	c := newClock()
	a := newAccess(t, c)
	tok, _, _ := a.Issue(authn.Principal{UserID: "u1", Roles: []string{"user"}})
	var b authn.Authenticator = token.NewBearer(a)

	req := func(h string) *http.Request {
		r := httptest.NewRequest("GET", "/me", nil)
		if h != "" {
			r.Header.Set("Authorization", h)
		}
		return r
	}

	p, err := b.Authenticate(req("Bearer " + tok))
	if err != nil || p.UserID != "u1" {
		t.Fatalf("got %+v, %v", p, err)
	}
	if _, err := b.Authenticate(req("bearer " + tok)); err != nil {
		t.Fatalf("scheme must be case-insensitive: %v", err)
	}
	for name, h := range map[string]string{
		"missing":      "",
		"basic":        "Basic dTE6cHc=",
		"no token":     "Bearer ",
		"no scheme":    tok,
		"bad token":    "Bearer nope",
		"extra fields": "Bearer " + tok + " junk",
	} {
		if _, err := b.Authenticate(req(h)); !errors.Is(err, authn.ErrUnauthenticated) {
			t.Errorf("%s: got %v, want ErrUnauthenticated", name, err)
		}
	}
}
