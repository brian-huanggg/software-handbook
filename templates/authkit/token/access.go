// Package token implements stateless access JWTs and rotating refresh tokens.
package token

import (
	"crypto/sha256"
	"encoding/hex"
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
	Secret []byte // HMAC key that signs new tokens, at least 32 bytes
	// PreviousSecrets still verify, but never sign. To rotate: make the new
	// key Secret, move the old one here, and drop it once the longest-lived
	// token signed with it has expired (one TTL later).
	PreviousSecrets [][]byte
	Issuer          string        // required; checked on verify
	TTL             time.Duration // default 15m
	Now             func() time.Time
}

// Access issues and verifies access JWTs. Every token names its key in the
// "kid" header, so verification picks the key directly instead of trying
// each one.
type Access struct {
	kid    string            // id of the signing key
	keys   map[string][]byte // kid -> key, signing key included
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
	keys := make(map[string][]byte, 1+len(o.PreviousSecrets))
	for i, k := range append([][]byte{o.Secret}, o.PreviousSecrets...) {
		if len(k) < minSecretLen {
			return nil, fmt.Errorf("token: secret %d must be at least %d bytes", i, minSecretLen)
		}
		id := keyID(k)
		if _, dup := keys[id]; dup {
			return nil, fmt.Errorf("token: secret %d is a duplicate", i)
		}
		keys[id] = append([]byte(nil), k...)
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
		kid:    keyID(o.Secret),
		keys:   keys,
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
	tok.Header["kid"] = a.kid
	s, err := tok.SignedString(a.keys[a.kid])
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return s, exp, nil
}

// Verify returns the Principal in a valid token. Every failure is reported
// as authn.ErrUnauthenticated; the cause is deliberately not exposed.
func (a *Access) Verify(s string) (authn.Principal, error) {
	var c claims
	_, err := a.parser.ParseWithClaims(s, &c, a.key)
	if err != nil || c.Subject == "" {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	return authn.Principal{UserID: c.Subject, Roles: c.Roles}, nil
}

// key returns the verification key the token's "kid" names. A token without
// a known kid is rejected rather than tried against every key.
func (a *Access) key(t *jwt.Token) (any, error) {
	kid, _ := t.Header["kid"].(string)
	k, ok := a.keys[kid]
	if !ok {
		return nil, errors.New("unknown kid")
	}
	return k, nil
}

// keyID derives a key's id from the key itself, so rotation needs no extra
// configuration. It is a truncated hash of a high-entropy secret, so it
// reveals nothing usable about the key.
func keyID(k []byte) string {
	sum := sha256.Sum256(append([]byte("authkit-kid:"), k...))
	return hex.EncodeToString(sum[:8])
}
