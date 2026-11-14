// Package token implements stateless access JWTs and rotating refresh tokens.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
)

const (
	minSecretLen = 32
	defaultTTL   = 15 * time.Minute
)

// signingMethod is the only accepted algorithm. Pinning it (instead of
// trusting the token's own "alg" header) blocks alg=none and algorithm
// confusion attacks.
var signingMethod = jwt.SigningMethodHS256

type AccessOptions struct {
	Secret []byte        // HMAC key, at least 32 bytes
	Issuer string        // required; checked on verify
	TTL    time.Duration // default 15m
	Now    func() time.Time
}

// Access issues and verifies access JWTs.
type Access struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
	parser *jwt.Parser
}

type claims struct {
	Roles []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

func NewAccess(o AccessOptions) (*Access, error) {
	if len(o.Secret) < minSecretLen {
		return nil, fmt.Errorf("token: secret must be at least %d bytes", minSecretLen)
	}
	if o.Issuer == "" {
		return nil, errors.New("token: issuer is required")
	}
	if o.TTL <= 0 {
		o.TTL = defaultTTL
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Access{
		secret: append([]byte(nil), o.Secret...),
		issuer: o.Issuer,
		ttl:    o.TTL,
		now:    o.Now,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{signingMethod.Alg()}),
			jwt.WithIssuer(o.Issuer),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
			jwt.WithTimeFunc(o.Now),
		),
	}, nil
}

// Issue signs a token for p and returns it with its expiry.
func (a *Access) Issue(p authn.Principal) (string, time.Time, error) {
	now := a.now()
	exp := now.Add(a.ttl)
	tok := jwt.NewWithClaims(signingMethod, claims{
		Roles: p.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.UserID,
			Issuer:    a.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	s, err := tok.SignedString(a.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return s, exp, nil
}

// Verify returns the Principal in a valid token. Every failure is reported
// as authn.ErrUnauthenticated; the cause is deliberately not exposed.
func (a *Access) Verify(s string) (authn.Principal, error) {
	var c claims
	_, err := a.parser.ParseWithClaims(s, &c, func(*jwt.Token) (any, error) { return a.secret, nil })
	if err != nil || c.Subject == "" {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	return authn.Principal{UserID: c.Subject, Roles: c.Roles}, nil
}
