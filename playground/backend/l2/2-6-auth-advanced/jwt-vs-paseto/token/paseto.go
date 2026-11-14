package token

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/json"
	"strings"
	"time"
)

// header is signed together with the payload, so a token cannot be replayed
// as another version or purpose.
const header = "v4.public."

// le64 encodes n as 8 little-endian bytes with the top bit cleared, as the
// PASETO spec requires.
func le64(n uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n&^(1<<63))
	return b
}

// pae is the Pre-Authentication Encoding: it joins the pieces so that no two
// different lists of pieces can produce the same bytes.
func pae(pieces ...[]byte) []byte {
	out := le64(uint64(len(pieces)))
	for _, p := range pieces {
		out = append(out, le64(uint64(len(p)))...)
		out = append(out, p...)
	}
	return out
}

// SignPublic issues a v4.public token. footer is authenticated but not
// encrypted. implicit is authenticated and never appears in the token: both
// sides must supply the same value (for example a tenant ID).
func SignPublic(priv ed25519.PrivateKey, c Claims, footer, implicit []byte) (string, error) {
	m, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, pae([]byte(header), m, footer, implicit))
	tok := header + enc.EncodeToString(append(m, sig...))
	if len(footer) > 0 {
		tok += "." + enc.EncodeToString(footer)
	}
	return tok, nil
}

// VerifyPublic returns the claims and the footer of a valid token. The footer
// is only trustworthy because the signature covered it.
func VerifyPublic(pub ed25519.PublicKey, tok, iss string, implicit []byte, now time.Time) (Claims, []byte, error) {
	rest, ok := strings.CutPrefix(tok, header)
	if !ok {
		return Claims{}, nil, ErrFormat
	}
	parts := strings.Split(rest, ".")
	if len(parts) > 2 {
		return Claims{}, nil, ErrFormat
	}
	body, err := enc.DecodeString(parts[0])
	if err != nil || len(body) <= ed25519.SignatureSize {
		return Claims{}, nil, ErrFormat
	}
	var footer []byte
	if len(parts) == 2 {
		if footer, err = enc.DecodeString(parts[1]); err != nil {
			return Claims{}, nil, ErrFormat
		}
	}
	m, sig := body[:len(body)-ed25519.SignatureSize], body[len(body)-ed25519.SignatureSize:]
	if !ed25519.Verify(pub, pae([]byte(header), m, footer, implicit), sig) {
		return Claims{}, nil, ErrSignature
	}
	var c Claims
	if err := json.Unmarshal(m, &c); err != nil {
		return Claims{}, nil, ErrFormat
	}
	if err := c.check(iss, now); err != nil {
		return Claims{}, nil, err
	}
	return c, footer, nil
}

// PeekPublic reads the payload WITHOUT verifying anything. It exists to show
// that a public token is signed, not encrypted.
func PeekPublic(tok string) (string, error) {
	rest, ok := strings.CutPrefix(tok, header)
	if !ok {
		return "", ErrFormat
	}
	body, err := enc.DecodeString(strings.SplitN(rest, ".", 2)[0])
	if err != nil || len(body) <= ed25519.SignatureSize {
		return "", ErrFormat
	}
	return string(body[:len(body)-ed25519.SignatureSize]), nil
}
