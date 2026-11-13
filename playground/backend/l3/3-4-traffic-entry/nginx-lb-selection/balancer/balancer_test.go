package balancer

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestRoundRobinIsSmooth(t *testing.T) {
	p := NewPool(Server{Name: "a", Weight: 5}, Server{Name: "b", Weight: 1}, Server{Name: "c", Weight: 1})

	var got []string
	for i := 0; i < 14; i++ {
		got = append(got, p.RoundRobin().Name)
	}
	// The same 7-request cycle twice, with b and c spread out instead of
	// five a's in a row.
	want := "a a b a c a a a a b a c a a"
	if s := strings.Join(got, " "); s != want {
		t.Fatalf("sequence = %q, want %q", s, want)
	}
}

func TestLeastConnPrefersLessLoadedPerWeight(t *testing.T) {
	p := NewPool(Server{Name: "small", Weight: 1}, Server{Name: "big", Weight: 3})

	// 2 conns on small (2/1) vs 5 on big (5/3): big is still the lighter one.
	for i := 0; i < 2; i++ {
		p.Acquire(p.servers[0])
	}
	for i := 0; i < 5; i++ {
		p.Acquire(p.servers[1])
	}
	if got := p.LeastConn().Name; got != "big" {
		t.Fatalf("LeastConn() = %s, want big", got)
	}

	// 6 conns on big (6/3 = 2) ties with small (2/1): the tie is shared by
	// weight, so over 4 requests big gets 3 and small gets 1.
	p.Acquire(p.servers[1])
	got := map[string]int{}
	for i := 0; i < 4; i++ {
		got[p.LeastConn().Name]++
	}
	if got["big"] != 3 || got["small"] != 1 {
		t.Fatalf("tie shared as %v, want big=3 small=1", got)
	}

	p.Release(p.servers[0])
	if got := p.LeastConn().Name; got != "small" {
		t.Fatalf("after Release, LeastConn() = %s, want small", got)
	}
}

func TestRandomTwoNeverPicksTheBusiestOfThree(t *testing.T) {
	p := NewPool(Server{Name: "a", Weight: 1}, Server{Name: "b", Weight: 1}, Server{Name: "c", Weight: 1})
	for i := 0; i < 10; i++ {
		p.Acquire(p.servers[2]) // c is the clear loser
	}

	rnd := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 1000; i++ {
		if got := p.RandomTwo(rnd).Name; got == "c" {
			t.Fatalf("iteration %d picked the busiest server", i)
		}
	}
}

func TestIPHashKey(t *testing.T) {
	tests := []struct{ ip, want string }{
		{"203.0.113.7", "203.0.113"},
		{"203.0.113.250", "203.0.113"},
		{"2001:db8::1", "2001:db8::1"},
	}
	for _, tt := range tests {
		if got := IPHashKey(tt.ip); got != tt.want {
			t.Errorf("IPHashKey(%q) = %q, want %q", tt.ip, got, tt.want)
		}
	}
}

func keys(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("user-%d", i)
	}
	return out
}

func TestRingOnlyMovesKeysOfTheRemovedServer(t *testing.T) {
	before := NewRing([]string{"a", "b", "c"}, 160)
	after := NewRing([]string{"a", "b"}, 160)

	for _, k := range keys(5000) {
		was, now := before.Get(k), after.Get(k)
		if was != "c" && was != now {
			t.Fatalf("key %s moved from %s to %s although %s stayed", k, was, now, was)
		}
	}
}

func TestModPickMovesMostKeysWhenServerCountChanges(t *testing.T) {
	three := []string{"a", "b", "c"}
	four := []string{"a", "b", "c", "d"}

	moved := 0
	ks := keys(5000)
	for _, k := range ks {
		if ModPick(k, three) != ModPick(k, four) {
			moved++
		}
	}
	// Going from 3 to 4 servers moves about 3 in 4 keys under mod.
	if ratio := float64(moved) / float64(len(ks)); ratio < 0.6 {
		t.Fatalf("moved ratio = %.2f, want at least 0.60", ratio)
	}
}
