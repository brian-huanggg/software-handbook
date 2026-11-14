// Package jwt is a minimal HS256 JWT implementation for teaching.
// In production, use a maintained library such as github.com/golang-jwt/jwt.
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMalformed = errors.New("jwt: malformed token")
	ErrAlgorithm = errors.New("jwt: unexpected algorithm")
	ErrSignature = errors.New("jwt: bad signature")
	ErrExpired   = errors.New("jwt: expired")
	ErrIssuer    = errors.New("jwt: unexpected issuer")
)

// Claims holds the registered claims this example uses (RFC 7519 section 4.1).
type Claims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

var enc = base64.RawURLEncoding

func mac(key []byte, signingInput string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(signingInput))
	return h.Sum(nil)
}

// Sign returns header.payload.signature, each part base64url-encoded.
func Sign(c Claims, key []byte) (string, error) {
	hb, err := json.Marshal(header{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", fmt.Errorf("jwt: encode header: %w", err)
	}
	pb, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("jwt: encode claims: %w", err)
	}
	input := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
	return input + "." + enc.EncodeToString(mac(key, input)), nil
}

// Verify checks the signature first, then the claims. The algorithm is fixed
// by the server, never taken from the token's own header.
func Verify(token string, key []byte, wantIssuer string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformed
	}
	hb, err := enc.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var h header
	if err := json.Unmarshal(hb, &h); err != nil {
		return Claims{}, ErrMalformed
	}
	if h.Alg != "HS256" {
		return Claims{}, ErrAlgorithm
	}
	sig, err := enc.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	if !hmac.Equal(sig, mac(key, parts[0]+"."+parts[1])) {
		return Claims{}, ErrSignature
	}
	pb, err := enc.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var c Claims
	if err := json.Unmarshal(pb, &c); err != nil {
		return Claims{}, ErrMalformed
	}
	// A token without exp must not live forever.
	if c.ExpiresAt == 0 || !now.Before(time.Unix(c.ExpiresAt, 0)) {
		return Claims{}, ErrExpired
	}
	if c.Issuer != wantIssuer {
		return Claims{}, ErrIssuer
	}
	return c, nil
}

// Peek decodes the payload WITHOUT verifying anything. It exists to show
// that a JWT payload is encoded, not encrypted.
func Peek(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", ErrMalformed
	}
	b, err := enc.DecodeString(parts[1])
	if err != nil {
		return "", ErrMalformed
	}
	return string(b), nil
}
