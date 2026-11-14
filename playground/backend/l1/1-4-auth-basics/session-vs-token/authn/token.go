package authn

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var enc = base64.RawURLEncoding

type claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

// Token is stateless: the credential carries the user and an expiry, and an
// HMAC proves the server issued it. The server stores nothing per login.
// It has the same shape as a JWT but skips the header; use a library in
// production (see the JWT article for what a real verifier must check).
type Token struct {
	key []byte
	ttl time.Duration
	now func() time.Time // replaced in tests
}

// NewToken returns a token issuer whose tokens live for ttl.
func NewToken(key []byte, ttl time.Duration) *Token {
	return &Token{key: key, ttl: ttl, now: time.Now}
}

func (t *Token) mac(payload string) []byte {
	h := hmac.New(sha256.New, t.key)
	h.Write([]byte(payload))
	return h.Sum(nil)
}

func (t *Token) Login(userID string) (string, error) {
	b, err := json.Marshal(claims{Sub: userID, Exp: t.now().Add(t.ttl).Unix()})
	if err != nil {
		return "", fmt.Errorf("authn: encode claims: %w", err)
	}
	payload := enc.EncodeToString(b)
	return payload + "." + enc.EncodeToString(t.mac(payload)), nil
}

func (t *Token) Authenticate(tok string) (string, error) {
	payload, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return "", ErrInvalid
	}
	got, err := enc.DecodeString(sig)
	if err != nil || !hmac.Equal(got, t.mac(payload)) {
		return "", ErrInvalid
	}
	b, err := enc.DecodeString(payload)
	if err != nil {
		return "", ErrInvalid
	}
	var c claims
	if err := json.Unmarshal(b, &c); err != nil {
		return "", ErrInvalid
	}
	// A token without exp must not live forever.
	if c.Exp == 0 || !t.now().Before(time.Unix(c.Exp, 0)) {
		return "", ErrInvalid
	}
	return c.Sub, nil
}

// Logout cannot do anything: there is no server-side record to delete, so
// every copy of the token stays valid until it expires.
func (t *Token) Logout(string) error { return nil }
