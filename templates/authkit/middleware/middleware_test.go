package middleware_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
	"github.com/brian-huanggg/software-handbook/templates/authkit/token"
)

type stub struct {
	p   authn.Principal
	err error
}

func (s stub) Authenticate(*http.Request) (authn.Principal, error) { return s.p, s.err }

func serve(a authn.Authenticator) (*httptest.ResponseRecorder, *bool, *authn.Principal) {
	called := false
	var seen authn.Principal
	h := middleware.Require(a)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		seen, _ = authn.FromContext(r.Context())
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/me", nil))
	return w, &called, &seen
}

func TestRequire_PutsPrincipalInContext(t *testing.T) {
	want := authn.Principal{UserID: "u1", Roles: []string{"user"}}
	w, called, seen := serve(stub{p: want})
	if w.Code != 200 || !*called {
		t.Fatalf("code=%d called=%v", w.Code, *called)
	}
	if !reflect.DeepEqual(*seen, want) {
		t.Fatalf("principal = %+v, want %+v", *seen, want)
	}
}

func TestRequire_Unauthenticated_401(t *testing.T) {
	for name, err := range map[string]error{
		"plain":   authn.ErrUnauthenticated,
		"wrapped": fmt.Errorf("session: %w", authn.ErrUnauthenticated),
	} {
		w, called, _ := serve(stub{err: err})
		if w.Code != http.StatusUnauthorized || *called {
			t.Errorf("%s: code=%d called=%v, want 401 and next not called", name, w.Code, *called)
		}
	}
}

func TestRequire_InfraError_500_NoLeak(t *testing.T) {
	w, called, _ := serve(stub{err: errors.New("pq: password authentication failed for user authkit")})
	if w.Code != http.StatusInternalServerError || *called {
		t.Fatalf("code=%d called=%v, want 500 and next not called", w.Code, *called)
	}
	if strings.Contains(w.Body.String(), "pq") || strings.Contains(w.Body.String(), "authkit") {
		t.Fatalf("internal error leaked: %q", w.Body.String())
	}
}

// The same middleware works for any Authenticator; here, the bearer one.
func TestRequire_WithBearer(t *testing.T) {
	a, _ := token.NewAccess(token.AccessOptions{Secret: []byte(strings.Repeat("k", 32)), Issuer: "i", TTL: time.Minute})
	tok, _, _ := a.Issue(authn.Principal{UserID: "u9"})
	h := middleware.Require(token.NewBearer(a))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := authn.FromContext(r.Context())
		fmt.Fprint(w, p.UserID)
	}))

	r := httptest.NewRequest("GET", "/me", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "u9" {
		t.Fatalf("code=%d body=%q", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/me", nil))
	if w.Code != 401 {
		t.Fatalf("no header: code=%d, want 401", w.Code)
	}
}
