// Command nginx-lb-selection prints how nginx's selection algorithms behave.
package main

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"

	"handbook/playground/golang/backend/nginx-lb-selection/balancer"
)

func main() {
	roundRobin()
	leastConn()
	moveOnChange()
	ipHashBehindNAT()
}

func roundRobin() {
	fmt.Println("== round robin, weights a=5 b=1 c=1 ==")
	p := balancer.NewPool(
		balancer.Server{Name: "a", Weight: 5},
		balancer.Server{Name: "b", Weight: 1},
		balancer.Server{Name: "c", Weight: 1},
	)
	var seq []string
	for i := 0; i < 7; i++ {
		seq = append(seq, p.RoundRobin().Name)
	}
	fmt.Println("first 7 requests:", strings.Join(seq, " "))
	fmt.Println()
}

// leastConn sends 3,000 requests, one per tick, to a pool where one server
// is 10x slower. A slow server keeps each request in flight for longer, so
// its connection count stays high.
func leastConn() {
	fmt.Println("== slow server: round robin vs least_conn vs random two ==")
	type picker func(p *balancer.Pool) *balancer.Server
	rnd := rand.New(rand.NewPCG(7, 7))
	strategies := []struct {
		name string
		pick picker
	}{
		{"round_robin", func(p *balancer.Pool) *balancer.Server { return p.RoundRobin() }},
		{"least_conn", func(p *balancer.Pool) *balancer.Server { return p.LeastConn() }},
		{"random two", func(p *balancer.Pool) *balancer.Server { return p.RandomTwo(rnd) }},
	}
	for _, st := range strategies {
		p := balancer.NewPool(
			balancer.Server{Name: "fast-1", Weight: 1},
			balancer.Server{Name: "fast-2", Weight: 1},
			balancer.Server{Name: "slow", Weight: 1},
		)
		picked := simulate(p, st.pick)
		fmt.Printf("%-12s fast-1=%4d fast-2=%4d slow=%4d\n", st.name, picked["fast-1"], picked["fast-2"], picked["slow"])
	}
	fmt.Println()
}

// simulate runs a discrete-time loop: one request arrives per tick, a fast
// server finishes a request after 2 ticks, the slow one after 20.
func simulate(p *balancer.Pool, pick func(*balancer.Pool) *balancer.Server) map[string]int {
	type inflight struct {
		s    *balancer.Server
		done int
	}
	var running []inflight
	picked := map[string]int{}
	for tick := 0; tick < 3000; tick++ {
		kept := running[:0]
		for _, r := range running {
			if r.done <= tick {
				p.Release(r.s)
			} else {
				kept = append(kept, r)
			}
		}
		running = kept

		s := pick(p)
		p.Acquire(s)
		picked[s.Name]++
		cost := 2
		if s.Name == "slow" {
			cost = 20
		}
		running = append(running, inflight{s, tick + cost})
	}
	return picked
}

func moveOnChange() {
	fmt.Println("== adding a 4th server: how many of 10,000 users change server ==")
	three := []string{"a", "b", "c"}
	four := []string{"a", "b", "c", "d"}
	ringThree := balancer.NewRing(three, 160)
	ringFour := balancer.NewRing(four, 160)

	modMoved, ringMoved := 0, 0
	for i := 0; i < 10000; i++ {
		k := fmt.Sprintf("user-%d", i)
		if balancer.ModPick(k, three) != balancer.ModPick(k, four) {
			modMoved++
		}
		if ringThree.Get(k) != ringFour.Get(k) {
			ringMoved++
		}
	}
	fmt.Printf("hash (mod):        %5d moved\n", modMoved)
	fmt.Printf("hash consistent:   %5d moved\n", ringMoved)
	fmt.Println()
}

// ipHashBehindNAT puts 1,000 users behind one office NAT (one /24) and
// 1,000 more spread over many networks.
func ipHashBehindNAT() {
	fmt.Println("== ip_hash: 1,000 users behind one NAT vs 1,000 spread out ==")
	servers := []string{"a", "b", "c"}

	nat := map[string]int{}
	for i := 0; i < 1000; i++ {
		ip := fmt.Sprintf("203.0.113.%d", i%250+1)
		nat[balancer.ModPick(balancer.IPHashKey(ip), servers)]++
	}
	spread := map[string]int{}
	for i := 0; i < 1000; i++ {
		ip := fmt.Sprintf("%d.%d.%d.9", 10+i%200, i%251, i%241)
		spread[balancer.ModPick(balancer.IPHashKey(ip), servers)]++
	}
	fmt.Println("behind one NAT:", format(nat))
	fmt.Println("spread out:    ", format(spread))
}

func format(m map[string]int) string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", n, m[n]))
	}
	return strings.Join(parts, " ")
}
