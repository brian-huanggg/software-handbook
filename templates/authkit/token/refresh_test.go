package token_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"authkit/token"
)

type rfix struct {
	m     *token.RefreshManager
	store token.RefreshStore
	c     *clock
}

func newRefresh(t *testing.T) *rfix {
	t.Helper()
	f := &rfix{store: token.NewMemoryRefreshStore(), c: newClock()}
	f.m = token.NewRefreshManager(f.store, token.RefreshOptions{TTL: 24 * time.Hour, Now: f.c.now})
	return f
}

var ctx = context.Background()

func TestRefresh_ReuseErrorIsAnInvalidRefresh(t *testing.T) {
	if !errors.Is(token.ErrReuseDetected, token.ErrInvalidRefresh) {
		t.Fatal("ErrReuseDetected must satisfy errors.Is(_, ErrInvalidRefresh)")
	}
}

func TestRefresh_StartAndRotateChain(t *testing.T) {
	f := newRefresh(t)
	t0, err := f.m.Start(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(t0) != 43 {
		t.Fatalf("token length %d, want 43", len(t0))
	}

	t1, uid, err := f.m.Rotate(ctx, t0)
	if err != nil || uid != "u1" || t1 == t0 {
		t.Fatalf("Rotate = %q, %q, %v", t1, uid, err)
	}
	t2, uid, err := f.m.Rotate(ctx, t1)
	if err != nil || uid != "u1" || t2 == t1 || t2 == t0 {
		t.Fatalf("second Rotate = %q, %q, %v", t2, uid, err)
	}
}

func TestRefresh_StoreHoldsHashNotRawToken(t *testing.T) {
	f := newRefresh(t)
	raw, _ := f.m.Start(ctx, "u1")
	if _, err := f.store.Get(ctx, []byte(raw)); !errors.Is(err, token.ErrNotFound) {
		t.Fatal("raw token must not be usable as a store key")
	}
}

func TestRefresh_ReuseKillsWholeFamily(t *testing.T) {
	f := newRefresh(t)
	t0, _ := f.m.Start(ctx, "u1")
	t1, _, _ := f.m.Rotate(ctx, t0)

	// attacker (or buggy client) replays the already-used t0
	if _, _, err := f.m.Rotate(ctx, t0); !errors.Is(err, token.ErrReuseDetected) {
		t.Fatalf("replay: got %v, want ErrReuseDetected", err)
	}
	// the newest legitimate token is now dead too
	if _, _, err := f.m.Rotate(ctx, t1); !errors.Is(err, token.ErrInvalidRefresh) {
		t.Fatalf("latest token after reuse: got %v, want ErrInvalidRefresh", err)
	}
}

func TestRefresh_ReuseDoesNotAffectOtherFamilies(t *testing.T) {
	f := newRefresh(t)
	a0, _ := f.m.Start(ctx, "u1")
	b0, _ := f.m.Start(ctx, "u1") // second device
	_, _, _ = f.m.Rotate(ctx, a0)
	_, _, _ = f.m.Rotate(ctx, a0) // reuse on family A

	if _, _, err := f.m.Rotate(ctx, b0); err != nil {
		t.Fatalf("family B must survive: %v", err)
	}
}

func TestRefresh_Rejects(t *testing.T) {
	f := newRefresh(t)
	for name, raw := range map[string]string{
		"empty":   "",
		"garbage": "!!!",
		"unknown": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		if _, _, err := f.m.Rotate(ctx, raw); !errors.Is(err, token.ErrInvalidRefresh) || errors.Is(err, token.ErrReuseDetected) {
			t.Errorf("%s: got %v, want plain ErrInvalidRefresh", name, err)
		}
	}
}

func TestRefresh_Expired(t *testing.T) {
	f := newRefresh(t)
	t0, _ := f.m.Start(ctx, "u1")
	f.c.t = f.c.t.Add(24*time.Hour - time.Second)
	t1, _, err := f.m.Rotate(ctx, t0)
	if err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	f.c.t = f.c.t.Add(25 * time.Hour)
	if _, _, err := f.m.Rotate(ctx, t1); !errors.Is(err, token.ErrInvalidRefresh) {
		t.Fatalf("after expiry: got %v", err)
	}
}

func TestRefresh_Revoke(t *testing.T) {
	f := newRefresh(t)
	t0, _ := f.m.Start(ctx, "u1")
	t1, _, _ := f.m.Rotate(ctx, t0)

	if err := f.m.Revoke(ctx, t1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.m.Rotate(ctx, t1); !errors.Is(err, token.ErrInvalidRefresh) {
		t.Fatalf("revoked token rotated: %v", err)
	}
	if err := f.m.Revoke(ctx, "unknown"); err != nil {
		t.Fatalf("revoking unknown token should be a no-op: %v", err)
	}
	if err := f.m.Revoke(ctx, ""); err != nil {
		t.Fatalf("revoking empty token should be a no-op: %v", err)
	}
}

// Two requests presenting the same token at once: only one may rotate.
func TestRefresh_ConcurrentRotateHasOneWinner(t *testing.T) {
	f := newRefresh(t)
	t0, _ := f.m.Start(ctx, "u1")

	var wins, reuse, invalid atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := f.m.Rotate(ctx, t0)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, token.ErrReuseDetected):
				reuse.Add(1)
			case errors.Is(err, token.ErrInvalidRefresh):
				// family already deleted by an earlier loser
				invalid.Add(1)
			}
		}()
	}
	wg.Wait()
	w, r, i := wins.Load(), reuse.Load(), invalid.Load()
	if w != 1 || r < 1 || w+r+i != 20 {
		t.Fatalf("wins=%d reuse=%d invalid=%d, want 1 winner, >=1 reuse, 20 total", w, r, i)
	}
}
