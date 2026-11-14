//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/verify.go` to compile-check only.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var enc = base64.RawURLEncoding

type footer struct {
	KID string `json:"kid"`
}

type claims struct {
	Sub string `json:"sub"`
	Exp string `json:"exp"`
}

// verifyPublic checks a v4.public token and returns the user ID.
// keys maps a key ID to the public key that may verify tokens carrying it.
func verifyPublic(tok string, keys map[string]ed25519.PublicKey) (string, error) {
	parts := strings.Split(tok, ".")
	body, err := enc.DecodeString(parts[2])
	if err != nil || len(body) <= ed25519.SignatureSize {
		return "", errors.New("malformed")
	}
	msg, sig := body[:len(body)-ed25519.SignatureSize], body[len(body)-ed25519.SignatureSize:]

	fb, err := enc.DecodeString(parts[3])
	if err != nil {
		return "", errors.New("malformed footer")
	}
	var f footer
	if err := json.Unmarshal(fb, &f); err != nil {
		return "", errors.New("malformed footer")
	}
	pub, ok := keys[f.KID]
	if !ok {
		return "", errors.New("unknown key")
	}

	if !ed25519.Verify(pub, msg, sig) {
		return "", errors.New("bad signature")
	}

	var c claims
	if err := json.Unmarshal(msg, &c); err != nil {
		return "", errors.New("malformed payload")
	}
	return c.Sub, nil
}
