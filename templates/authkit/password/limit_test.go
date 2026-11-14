package password_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brian-huanggg/software-handbook/templates/authkit/password"
)

// slowHasher records the peak number of calls running at the same time.
type slowHasher struct{ running, peak atomic.Int32 }

func (h *slowHasher) enter() {
	n := h.running.Add(1)
	for {
		p := h.peak.Load()
		if n <= p || h.peak.CompareAndSwap(p, n) {
			break
		}
	}
	time.Sleep(5 * time.Millisecond)
	h.running.Add(-1)
}
func (h *slowHasher) Hash(string) (string, error)         { h.enter(); return "x", nil }
func (h *slowHasher) Verify(string, string) (bool, error) { h.enter(); return true, nil }

func TestLimited_CapsConcurrency(t *testing.T) {
	inner := &slowHasher{}
	h := password.NewLimited(inner, 2)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, _ = h.Hash("pw")
			} else {
				_, _ = h.Verify("pw", "x")
			}
		}(i)
	}
	wg.Wait()
	if p := inner.peak.Load(); p > 2 {
		t.Fatalf("peak concurrency %d, want <= 2", p)
	}
}

func TestLimited_PassesThroughResults(t *testing.T) {
	h := password.NewLimited(password.NewArgon2id(nil), 1)
	enc, err := h.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Verify("correct horse", enc); err != nil || !ok {
		t.Fatalf("Verify = %v, %v", ok, err)
	}
	if ok, _ := h.Verify("wrong", enc); ok {
		t.Fatal("wrong password verified")
	}
}
