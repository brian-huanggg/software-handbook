package authz_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
	"github.com/brian-huanggg/software-handbook/templates/authkit/authz"
	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
)

type stub struct {
	p   authn.Principal
	err error
}

func (s stub) Authenticate(*http.Request) (authn.Principal, error) { return s.p, s.err }

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })

func do(h http.Handler, r *http.Request) int {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func withPrincipal(roles ...string) *http.Request {
	r := httptest.NewRequest("GET", "/admin", nil)
	return r.WithContext(authn.WithPrincipal(r.Context(), authn.Principal{UserID: "u1", Roles: roles}))
}

func TestRequireRole(t *testing.T) {
	admin := authz.RequireRole("admin")(ok)
	for name, c := range map[string]struct {
		h    http.Handler
		r    *http.Request
		want int
	}{
		"has role":       {admin, withPrincipal("admin"), 200},
		"has role among": {admin, withPrincipal("user", "admin"), 200},
		"lacks role":     {admin, withPrincipal("user"), 403},
		"no roles":       {admin, withPrincipal(), 403},
		"case-sensitive": {admin, withPrincipal("Admin"), 403},
		"no principal":   {admin, httptest.NewRequest("GET", "/admin", nil), 401},
		"any-of: first":  {authz.RequireRole("admin", "editor")(ok), withPrincipal("admin"), 200},
		"any-of: second": {authz.RequireRole("admin", "editor")(ok), withPrincipal("editor"), 200},
		"any-of: none":   {authz.RequireRole("admin", "editor")(ok), withPrincipal("user"), 403},
	} {
		if got := do(c.h, c.r); got != c.want {
			t.Errorf("%s: code=%d, want %d", name, got, c.want)
		}
	}
}

func TestRequireRole_NoRolesConfiguredPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic: a route guarded by nothing must not silently open")
		}
	}()
	authz.RequireRole()
}

func TestHasRole(t *testing.T) {
	p := authn.Principal{Roles: []string{"user", "admin"}}
	if !authz.HasRole(p, "admin") || authz.HasRole(p, "root") || authz.HasRole(authn.Principal{}, "admin") {
		t.Fatal("HasRole wrong")
	}
}

// 401 and 403 must never be mixed: not logged in is 401 even on an admin route.
func TestChain_401Vs403(t *testing.T) {
	chain := func(a authn.Authenticator) http.Handler {
		return middleware.Require(a)(authz.RequireRole("admin")(ok))
	}
	req := func() *http.Request { return httptest.NewRequest("GET", "/admin", nil) }

	if got := do(chain(stub{err: authn.ErrUnauthenticated}), req()); got != 401 {
		t.Errorf("anonymous: %d, want 401", got)
	}
	if got := do(chain(stub{p: authn.Principal{UserID: "u", Roles: []string{"user"}}}), req()); got != 403 {
		t.Errorf("logged-in non-admin: %d, want 403", got)
	}
	if got := do(chain(stub{p: authn.Principal{UserID: "a", Roles: []string{"admin"}}}), req()); got != 200 {
		t.Errorf("admin: %d, want 200", got)
	}
}
