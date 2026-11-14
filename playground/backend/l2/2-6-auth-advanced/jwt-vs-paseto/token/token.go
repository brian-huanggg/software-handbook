// Package token puts a hand-rolled EdDSA JWT next to a PASETO v4.public token
// so the structural difference is visible. Teaching code: in production use a
// maintained library instead of rolling your own.
package token

import (
	"encoding/base64"
	"errors"
	"time"
)

// enc is the unpadded base64url both formats use.
var enc = base64.RawURLEncoding

var (
	ErrFormat    = errors.New("token: malformed")
	ErrAlgorithm = errors.New("token: algorithm not allowed")
	ErrSignature = errors.New("token: bad signature")
	ErrExpired   = errors.New("token: expired")
	ErrIssuer    = errors.New("token: wrong issuer")
)

// Claims is what both formats carry. time.Time marshals to RFC 3339, which is
// the format PASETO uses for exp; the JWT side converts to Unix seconds.
type Claims struct {
	Issuer    string    `json:"iss"`
	Subject   string    `json:"sub"`
	ExpiresAt time.Time `json:"exp"`
}

func (c Claims) check(iss string, now time.Time) error {
	if c.Issuer != iss {
		return ErrIssuer
	}
	// A token without exp must not live forever.
	if c.ExpiresAt.IsZero() || !now.Before(c.ExpiresAt) {
		return ErrExpired
	}
	return nil
}
