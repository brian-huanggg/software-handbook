// Package middleware adapts an authn.Authenticator to net/http.
package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
)

// Require authenticates every request with a. On success the Principal is
// stored in the request context (read it with authn.FromContext). A missing
// or invalid credential gets 401; any other error is an infrastructure
// failure and gets 500 with a generic body, so internals never leak.
//
// It only answers "who are you". Whether that principal may do something is
// authz's job, which answers 403.
func Require(a authn.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := a.Authenticate(r)
			switch {
			case err == nil:
				next.ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), p)))
			case errors.Is(err, authn.ErrUnauthenticated):
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
			default:
				slog.ErrorContext(r.Context(), "authenticate failed", "err", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		})
	}
}
