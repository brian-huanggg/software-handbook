package authn

import (
	"context"
	"errors"
	"net/http"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type Principal struct {
	UserID string
	Roles  []string
}

type Authenticator interface {
	Authenticate(r *http.Request) (Principal, error)
}

type ctxKey struct{}

// WithPrincipal returns a copy of ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the Principal stored by WithPrincipal.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
