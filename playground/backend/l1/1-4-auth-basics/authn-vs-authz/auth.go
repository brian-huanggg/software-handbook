package main

import (
	"context"
	"net/http"
	"strings"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID   string
	Role Role
}

type ctxKey struct{}

// Authenticator maps an opaque bearer token to a user. A real service
// would look up a session or verify a JWT here; the next articles cover both.
type Authenticator struct {
	users map[string]User
}

// Middleware answers one question only: who is calling? It never decides
// what the caller may do.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			unauthorized(w)
			return
		}
		user, ok := a.users[token]
		if !ok {
			unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
