// Package middleware adapts an authn.Authenticator to net/http.
package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
	"github.com/brian-huanggg/software-handbook/templates/authkit/respond"
)

// Require authenticates every request with a. On success the Principal is
// stored in the request context (read it with authn.FromContext). A missing
// or invalid credential gets 401; any other error is an infrastructure
// failure and gets 500 with a generic body, so internals never leak.
//
// It only answers "who are you". Whether that principal may do something is
// authz's job, which answers 403.
//
// If a also implements Challenger, a 401 carries its WWW-Authenticate
// challenge (RFC 9110 requires one on every 401; RFC 6750 defines "Bearer").
func Require(a authn.Authenticator) func(http.Handler) http.Handler {
	challenge := ""
	if c, ok := a.(Challenger); ok {
		challenge = c.Challenge()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := a.Authenticate(r)
			switch {
			case err == nil:
				next.ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), p)))
			case errors.Is(err, authn.ErrUnauthenticated):
				if challenge != "" {
					w.Header().Set("WWW-Authenticate", challenge)
				}
				respond.Error(w, http.StatusUnauthorized, "unauthenticated")
			default:
				slog.ErrorContext(r.Context(), "authenticate failed", "err", err)
				respond.Error(w, http.StatusInternalServerError, "internal server error")
			}
		})
	}
}

// Challenger is implemented by authenticators whose credential scheme has a
// WWW-Authenticate challenge. Cookie sessions have none.
type Challenger interface {
	Challenge() string
}

// NoStore marks every response uncacheable. Auth responses carry tokens,
// cookies and per-user data, and none of them is safe in a shared cache.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
