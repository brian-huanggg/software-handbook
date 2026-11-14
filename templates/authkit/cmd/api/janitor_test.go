package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunJanitor_RunsAtStartAndOnTickUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var ok, failing atomic.Int32
	done := make(chan struct{})
	go func() {
		runJanitor(ctx, 5*time.Millisecond,
			purger{"ok", func(context.Context) (int64, error) { ok.Add(1); return 1, nil }},
			// A failing purger must not stop the others.
			purger{"failing", func(context.Context) (int64, error) { failing.Add(1); return 0, errors.New("db down") }},
		)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for ok.Load() < 3 || failing.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("purgers ran %d/%d times, want >= 3 each", ok.Load(), failing.Load())
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runJanitor did not return after cancel")
	}
}
