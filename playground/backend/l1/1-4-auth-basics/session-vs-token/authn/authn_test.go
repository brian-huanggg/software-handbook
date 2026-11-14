package authn

import (
	"errors"
	"testing"
	"time"
)

var key = []byte("test-only-secret-test-only-secret")

func TestSessionLogoutIsImmediate(t *testing.T) {
	s := NewSession(time.Hour)
	id, err := s.Login("42")
	if err != nil {
		t.Fatal(err)
	}
	if u, err := s.Authenticate(id); err != nil || u != "42" {
		t.Fatalf("Authenticate = %q, %v; want 42, nil", u, err)
	}
	if err := s.Logout(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(id); !errors.Is(err, ErrInvalid) {
		t.Fatalf("after logout err = %v; want ErrInvalid", err)
	}
}

func TestSessionReadsStoreEveryRequest(t *testing.T) {
	s := NewSession(time.Hour)
	id, _ := s.Login("42")
	for range 5 {
		if _, err := s.Authenticate(id); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.Lookups(); got != 5 {
		t.Fatalf("Lookups = %d; want 5", got)
	}
}

func TestTokenSurvivesLogout(t *testing.T) {
	tk := NewToken(key, 15*time.Minute)
	tok, err := tk.Login("42")
	if err != nil {
		t.Fatal(err)
	}
	if err := tk.Logout(tok); err != nil {
		t.Fatal(err)
	}
	// This is the limitation, not a bug: nothing on the server can be deleted.
	if u, err := tk.Authenticate(tok); err != nil || u != "42" {
		t.Fatalf("after logout Authenticate = %q, %v; want 42, nil", u, err)
	}
}

func TestTokenExpiresAndRejectsTampering(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tk := NewToken(key, 15*time.Minute)
	tk.now = func() time.Time { return now }
	tok, _ := tk.Login("42")

	other := NewToken([]byte("a-different-key-a-different-key!"), 15*time.Minute)
	other.now = tk.now
	if _, err := other.Authenticate(tok); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong key err = %v; want ErrInvalid", err)
	}

	now = now.Add(15 * time.Minute)
	if _, err := tk.Authenticate(tok); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expired err = %v; want ErrInvalid", err)
	}
}

func TestHybridLogoutBlocksRefreshNotAccess(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	h := NewHybrid(key, 5*time.Minute, 24*time.Hour)
	h.now = func() time.Time { return now }
	h.access.now = h.now

	p, err := h.Login("42")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Logout(p.Refresh); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Refresh(p.Refresh); !errors.Is(err, ErrInvalid) {
		t.Fatalf("refresh after logout err = %v; want ErrInvalid", err)
	}
	if _, err := h.Authenticate(p.Access); err != nil {
		t.Fatalf("access token should still work until it expires: %v", err)
	}
	now = now.Add(5 * time.Minute)
	if _, err := h.Authenticate(p.Access); !errors.Is(err, ErrInvalid) {
		t.Fatalf("access after expiry err = %v; want ErrInvalid", err)
	}
}

func TestHybridRefreshRotates(t *testing.T) {
	h := NewHybrid(key, 5*time.Minute, 24*time.Hour)
	p1, _ := h.Login("42")
	p2, err := h.Refresh(p1.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Refresh == p1.Refresh {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := h.Refresh(p1.Refresh); !errors.Is(err, ErrInvalid) {
		t.Fatalf("reusing old refresh token err = %v; want ErrInvalid", err)
	}
}
