package session

import (
	"errors"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestStore(idle time.Duration) (*Store, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	s := NewStore(idle)
	s.now = clk.now
	return s, clk
}

func TestIDsAreLongAndUnique(t *testing.T) {
	s, _ := newTestStore(time.Hour)
	seen := map[string]bool{}
	for range 1000 {
		sess, err := s.Create("u1")
		if err != nil {
			t.Fatal(err)
		}
		if len(sess.ID) != 43 { // 32 bytes, base64url without padding
			t.Fatalf("id length = %d, want 43", len(sess.ID))
		}
		if seen[sess.ID] {
			t.Fatalf("duplicate id %q", sess.ID)
		}
		seen[sess.ID] = true
	}
}

func TestIdleExpiry(t *testing.T) {
	s, clk := newTestStore(30 * time.Minute)
	sess, _ := s.Create("u1")
	clk.advance(31 * time.Minute)
	if _, err := s.Get(sess.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after idle timeout: err = %v, want ErrNotFound", err)
	}
}

func TestGetSlidesExpiry(t *testing.T) {
	s, clk := newTestStore(30 * time.Minute)
	sess, _ := s.Create("u1")
	for range 4 { // 80 minutes in total, but never idle for 30
		clk.advance(20 * time.Minute)
		if _, err := s.Get(sess.ID); err != nil {
			t.Fatalf("Get while active: %v", err)
		}
	}
}

func TestDeleteAndSweep(t *testing.T) {
	s, clk := newTestStore(time.Minute)
	a, _ := s.Create("u1")
	b, _ := s.Create("u2")
	s.Delete(a.ID)
	if _, err := s.Get(a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete: err = %v, want ErrNotFound", err)
	}
	clk.advance(2 * time.Minute)
	if n := s.Sweep(); n != 1 {
		t.Fatalf("Sweep removed %d, want 1", n)
	}
	if _, err := s.Get(b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Sweep: err = %v, want ErrNotFound", err)
	}
}
