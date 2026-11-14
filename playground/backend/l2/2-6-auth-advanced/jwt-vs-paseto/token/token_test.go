package token

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	t0   = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	priv = ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	pub  = priv.Public().(ed25519.PublicKey)
	good = Claims{Issuer: "sho.rt", Subject: "42", ExpiresAt: t0.Add(15 * time.Minute)}
)

func TestPAEVectors(t *testing.T) {
	// Vectors from the PASETO spec (PAE section).
	if got := pae(); !bytes.Equal(got, make([]byte, 8)) {
		t.Errorf("pae() = %x", got)
	}
	want := append(append(le64(1), le64(4)...), "test"...)
	if got := pae([]byte("test")); !bytes.Equal(got, want) {
		t.Errorf("pae(test) = %x, want %x", got, want)
	}
}

func TestJWTRoundTrip(t *testing.T) {
	tok, err := SignJWT(priv, good)
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyJWT(pub, tok, "sho.rt", t0)
	if err != nil || c.Subject != "42" {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestJWTRejectsOtherAlgorithms(t *testing.T) {
	tok, _ := SignJWT(priv, good)
	parts := strings.Split(tok, ".")
	for _, alg := range []string{"none", "HS256"} {
		h := enc.EncodeToString([]byte(`{"alg":"` + alg + `","typ":"JWT"}`))
		forged := h + "." + parts[1] + "." + parts[2]
		if _, err := VerifyJWT(pub, forged, "sho.rt", t0); !errors.Is(err, ErrAlgorithm) {
			t.Errorf("alg=%s: err = %v, want ErrAlgorithm", alg, err)
		}
	}
}

func TestJWTRejectsExpiredAndWrongIssuer(t *testing.T) {
	tok, _ := SignJWT(priv, good)
	if _, err := VerifyJWT(pub, tok, "sho.rt", t0.Add(time.Hour)); !errors.Is(err, ErrExpired) {
		t.Errorf("expired: err = %v", err)
	}
	if _, err := VerifyJWT(pub, tok, "other", t0); !errors.Is(err, ErrIssuer) {
		t.Errorf("issuer: err = %v", err)
	}
}

func TestPublicRoundTrip(t *testing.T) {
	tok, err := SignPublic(priv, good, []byte(`{"kid":"k1"}`), []byte("tenant-a"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tok, "v4.public.") {
		t.Fatalf("token = %q", tok)
	}
	c, footer, err := VerifyPublic(pub, tok, "sho.rt", []byte("tenant-a"), t0)
	if err != nil || c.Subject != "42" || string(footer) != `{"kid":"k1"}` {
		t.Fatalf("got %+v, %q, %v", c, footer, err)
	}
}

func TestPublicFooterIsAuthenticated(t *testing.T) {
	tok, _ := SignPublic(priv, good, []byte(`{"kid":"k1"}`), nil)
	i := strings.LastIndex(tok, ".")
	forged := tok[:i+1] + enc.EncodeToString([]byte(`{"kid":"k2"}`))
	if _, _, err := VerifyPublic(pub, forged, "sho.rt", nil, t0); !errors.Is(err, ErrSignature) {
		t.Errorf("err = %v, want ErrSignature", err)
	}
}

func TestPublicImplicitAssertionMismatch(t *testing.T) {
	tok, _ := SignPublic(priv, good, nil, []byte("tenant-a"))
	if _, _, err := VerifyPublic(pub, tok, "sho.rt", []byte("tenant-b"), t0); !errors.Is(err, ErrSignature) {
		t.Errorf("err = %v, want ErrSignature", err)
	}
}

func TestPublicRejectsTamperedPayload(t *testing.T) {
	tok, _ := SignPublic(priv, good, nil, nil)
	body, _ := enc.DecodeString(strings.TrimPrefix(tok, header))
	body[10] ^= 1
	forged := header + enc.EncodeToString(body)
	if _, _, err := VerifyPublic(pub, forged, "sho.rt", nil, t0); !errors.Is(err, ErrSignature) {
		t.Errorf("err = %v, want ErrSignature", err)
	}
}

func TestPublicRejectsOtherVersionOrPurpose(t *testing.T) {
	tok, _ := SignPublic(priv, good, nil, nil)
	for _, h := range []string{"v3.public.", "v4.local.", "v2.public."} {
		forged := h + strings.TrimPrefix(tok, header)
		if _, _, err := VerifyPublic(pub, forged, "sho.rt", nil, t0); !errors.Is(err, ErrFormat) {
			t.Errorf("%s: err = %v, want ErrFormat", h, err)
		}
	}
}

func TestPublicRejectsExpired(t *testing.T) {
	tok, _ := SignPublic(priv, good, nil, nil)
	if _, _, err := VerifyPublic(pub, tok, "sho.rt", nil, t0.Add(time.Hour)); !errors.Is(err, ErrExpired) {
		t.Errorf("err = %v, want ErrExpired", err)
	}
}

func TestPublicPayloadIsReadable(t *testing.T) {
	tok, _ := SignPublic(priv, good, nil, nil)
	got, err := PeekPublic(tok)
	if err != nil || !strings.Contains(got, `"sub":"42"`) {
		t.Fatalf("got %q, %v", got, err)
	}
}
