package jwt

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	key = []byte("0123456789abcdef0123456789abcdef")
	t0  = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

func newToken(t *testing.T, exp time.Time) string {
	t.Helper()
	tok, err := Sign(Claims{Issuer: "sho.rt", Subject: "42", IssuedAt: t0.Unix(), ExpiresAt: exp.Unix()}, key)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestRoundTrip(t *testing.T) {
	tok := newToken(t, t0.Add(15*time.Minute))
	c, err := Verify(tok, key, "sho.rt", t0.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != "42" {
		t.Fatalf("sub = %q, want 42", c.Subject)
	}
}

func TestPayloadIsReadableWithoutKey(t *testing.T) {
	tok := newToken(t, t0.Add(time.Hour))
	got, err := Peek(tok)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"sub":"42"`) {
		t.Fatalf("payload not readable: %s", got)
	}
}

func TestTamperedPayload(t *testing.T) {
	tok := newToken(t, t0.Add(time.Hour))
	forged, err := Sign(Claims{Issuer: "sho.rt", Subject: "1", ExpiresAt: t0.Add(time.Hour).Unix()}, []byte("attacker-key-attacker-key-attack"))
	if err != nil {
		t.Fatal(err)
	}
	// Real header and signature, forged payload.
	p, q := strings.Split(tok, "."), strings.Split(forged, ".")
	bad := p[0] + "." + q[1] + "." + p[2]
	if _, err := Verify(bad, key, "sho.rt", t0); !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestAlgNoneRejected(t *testing.T) {
	tok := newToken(t, t0.Add(time.Hour))
	p := strings.Split(tok, ".")
	none := enc.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) + "." + p[1] + "."
	if _, err := Verify(none, key, "sho.rt", t0); !errors.Is(err, ErrAlgorithm) {
		t.Fatalf("err = %v, want ErrAlgorithm", err)
	}
}

func TestExpired(t *testing.T) {
	tok := newToken(t, t0.Add(15*time.Minute))
	if _, err := Verify(tok, key, "sho.rt", t0.Add(15*time.Minute)); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
}

func TestMissingExpRejected(t *testing.T) {
	tok, err := Sign(Claims{Issuer: "sho.rt", Subject: "42"}, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(tok, key, "sho.rt", t0); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
}

func TestWrongKeyAndIssuer(t *testing.T) {
	tok := newToken(t, t0.Add(time.Hour))
	if _, err := Verify(tok, []byte("some-other-key-some-other-key-xx"), "sho.rt", t0); !errors.Is(err, ErrSignature) {
		t.Fatalf("wrong key: err = %v, want ErrSignature", err)
	}
	if _, err := Verify(tok, key, "other.example", t0); !errors.Is(err, ErrIssuer) {
		t.Fatalf("wrong issuer: err = %v, want ErrIssuer", err)
	}
}

func TestMalformed(t *testing.T) {
	for _, tok := range []string{"", "a.b", "a.b.c.d", "!!.!!.!!"} {
		if _, err := Verify(tok, key, "sho.rt", t0); !errors.Is(err, ErrMalformed) {
			t.Errorf("Verify(%q) = %v, want ErrMalformed", tok, err)
		}
	}
}
