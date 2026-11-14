package token

import (
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"time"
)

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

// SignJWT issues an EdDSA (Ed25519) JWT, RFC 7519 with RFC 8037 for the algorithm.
func SignJWT(priv ed25519.PrivateKey, c Claims) (string, error) {
	hb, err := json.Marshal(jwtHeader{Alg: "EdDSA", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(jwtClaims{c.Issuer, c.Subject, c.ExpiresAt.Unix()})
	if err != nil {
		return "", err
	}
	input := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
	return input + "." + enc.EncodeToString(ed25519.Sign(priv, []byte(input))), nil
}

// VerifyJWT checks the token in the order that matters: header, signature,
// then claims. The algorithm comes from this function, not from the token.
func VerifyJWT(pub ed25519.PublicKey, tok, iss string, now time.Time) (Claims, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return Claims{}, ErrFormat
	}
	hb, err := enc.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrFormat
	}
	var h jwtHeader
	if err := json.Unmarshal(hb, &h); err != nil {
		return Claims{}, ErrFormat
	}
	if h.Alg != "EdDSA" {
		return Claims{}, ErrAlgorithm
	}
	sig, err := enc.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(pub, []byte(parts[0]+"."+parts[1]), sig) {
		return Claims{}, ErrSignature
	}
	pb, err := enc.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrFormat
	}
	var jc jwtClaims
	if err := json.Unmarshal(pb, &jc); err != nil {
		return Claims{}, ErrFormat
	}
	c := Claims{jc.Issuer, jc.Subject, time.Unix(jc.ExpiresAt, 0)}
	if err := c.check(iss, now); err != nil {
		return Claims{}, err
	}
	return c, nil
}
