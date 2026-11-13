//go:build exercise

package exercise

import (
	"strings"
	"testing"
)

func sequence(servers []*Server, n int) string {
	var names []string
	for i := 0; i < n; i++ {
		names = append(names, Next(servers).Name)
	}
	return strings.Join(names, " ")
}

func TestNextSpreadsHeavyServer(t *testing.T) {
	servers := []*Server{{Name: "a", Weight: 5}, {Name: "b", Weight: 1}, {Name: "c", Weight: 1}}
	if got, want := sequence(servers, 7), "a a b a c a a"; got != want {
		t.Fatalf("sequence = %q, want %q", got, want)
	}
}

func TestNextEqualWeightsAlternate(t *testing.T) {
	servers := []*Server{{Name: "a", Weight: 1}, {Name: "b", Weight: 1}}
	if got, want := sequence(servers, 4), "a b a b"; got != want {
		t.Fatalf("sequence = %q, want %q", got, want)
	}
}

func TestNextEmpty(t *testing.T) {
	if got := Next(nil); got != nil {
		t.Fatalf("Next(nil) = %v, want nil", got)
	}
}
