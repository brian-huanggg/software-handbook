//go:build exercise

package exercise

import (
	"testing"
	"time"

	pkchoice "handbook/playground/golang/backend/primary-key-choice"
)

func TestParseV7RoundTrip(t *testing.T) {
	want, err := pkchoice.NewV7(time.UnixMilli(1_700_000_000_000))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseV7(want.String())
	if err != nil {
		t.Fatalf("ParseV7(%q): %v", want, err)
	}
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestParseV7Rejects(t *testing.T) {
	bad := map[string]string{
		"too short":   "0190a5f2-3c1e-7abc-8def",
		"not hex":     "zzzzzzzz-3c1e-7abc-8def-0123456789ab",
		"no dashes":   "0190a5f23c1e7abc8def0123456789ab",
		"version 4":   "0190a5f2-3c1e-4abc-8def-0123456789ab",
		"bad variant": "0190a5f2-3c1e-7abc-cdef-0123456789ab",
	}
	for name, s := range bad {
		if _, err := ParseV7(s); err == nil {
			t.Errorf("%s: ParseV7(%q) = nil error, want error", name, s)
		}
	}
}
