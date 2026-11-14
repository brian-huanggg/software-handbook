// Package authn puts three ways of answering "who is this request from?"
// side by side: a server-side session, a stateless signed token, and a
// short-lived token paired with a revocable refresh token.
package authn

import "errors"

// ErrInvalid covers unknown, tampered, revoked and expired credentials alike,
// so callers cannot tell an attacker which part was wrong.
var ErrInvalid = errors.New("authn: invalid or expired credential")

// Authenticator is the part a login system must provide, whatever it stores.
type Authenticator interface {
	// Login returns a credential the client sends with every request.
	Login(userID string) (string, error)
	// Authenticate returns the user a credential belongs to.
	Authenticate(credential string) (string, error)
	// Logout asks the server to stop accepting the credential.
	Logout(credential string) error
}
