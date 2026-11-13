//go:build exercise

package exercise

import (
	"reflect"
	"testing"
	"time"

	flags "handbook/playground/golang/backend/feature-flags"
)

func TestStale(t *testing.T) {
	now := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	set := flags.Set{
		"old":       {Enabled: true, Percent: 100, Expires: "2026-10-01"},
		"future":    {Enabled: true, Percent: 10, Expires: "2026-12-31"},
		"no-expiry": {Enabled: true, Percent: 5},
		"today":     {Enabled: true, Percent: 5, Expires: "2026-11-13"},
		"aaa-old":   {Enabled: false, Expires: "2026-01-01"},
	}
	got := Stale(set, now)
	want := []string{"aaa-old", "no-expiry", "old"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Stale = %v, want %v", got, want)
	}
}
