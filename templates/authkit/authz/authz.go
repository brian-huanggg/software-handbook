// Package authz decides what an authenticated principal may do.
package authz

import (
	"net/http"
	"slices"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
)

// HasRole reports whether p holds role (exact, case-sensitive match).
func HasRole(p authn.Principal, role string) bool {
	return slices.Contains(p.Roles, role)
}

// RequireRole allows the request only if the principal has at least one of
// roles. It expects middleware.Require to have run first:
//
//	no principal in context       -> 401 (fail closed)
//	principal without a role      -> 403
//
// It panics when called with no roles, so a misconfigured route fails at
// startup instead of silently allowing nobody (or everybody).
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	if len(roles) == 0 {
		panic("authz: RequireRole needs at least one role")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := authn.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
				return
			}
			for _, role := range roles {
				if HasRole(p, role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}
