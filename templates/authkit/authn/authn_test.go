package authn_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"authkit/authn"
)

func TestFromContext_Empty(t *testing.T) {
	if _, ok := authn.FromContext(context.Background()); ok {
		t.Fatal("expected ok=false on empty context")
	}
}

func TestWithPrincipal_RoundTrip(t *testing.T) {
	want := authn.Principal{UserID: "u1", Roles: []string{"admin"}}
	ctx := authn.WithPrincipal(context.Background(), want)

	got, ok := authn.FromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWithPrincipal_Overrides(t *testing.T) {
	ctx := authn.WithPrincipal(context.Background(), authn.Principal{UserID: "a"})
	ctx = authn.WithPrincipal(ctx, authn.Principal{UserID: "b"})
	got, _ := authn.FromContext(ctx)
	if got.UserID != "b" {
		t.Fatalf("got %q, want b", got.UserID)
	}
}

type stubAuth struct{ err error }

func (s stubAuth) Authenticate(*http.Request) (authn.Principal, error) {
	return authn.Principal{}, s.err
}

func TestErrUnauthenticated_Wrappable(t *testing.T) {
	var a authn.Authenticator = stubAuth{err: authn.ErrUnauthenticated}
	_, err := a.Authenticate(nil)
	if !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("errors.Is failed: %v", err)
	}
}
