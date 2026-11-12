// Package balancer re-implements, in memory, the selection algorithms that
// nginx offers for an upstream block. It moves no traffic: it only answers
// "which server gets the next request".
package balancer

import (
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Server is one "server" line of an nginx upstream block.
type Server struct {
	Name   string
	Weight int

	conns   int // requests in flight, what least_conn looks at
	current int // running score of smooth weighted round robin
}

// Pool is one upstream block.
type Pool struct {
	mu      sync.Mutex
	servers []*Server
}

// NewPool copies servers. A weight below 1 is treated as 1, as in nginx.
func NewPool(servers ...Server) *Pool {
	p := &Pool{}
	for _, s := range servers {
		s := s
		if s.Weight < 1 {
			s.Weight = 1
		}
		p.servers = append(p.servers, &s)
	}
	return p
}

// RoundRobin is nginx's default: smooth weighted round robin.
func (p *Pool) RoundRobin() *Server {
	p.mu.Lock()
	defer p.mu.Unlock()
	return smoothPick(p.servers)
}

// smoothPick adds each weight to its server's running score, picks the
// highest score, then subtracts the total weight from the winner.
// Subtracting is what spreads a heavy server's turns out instead of
// serving them back to back.
func smoothPick(servers []*Server) *Server {
	var best *Server
	total := 0
	for _, s := range servers {
		s.current += s.Weight
		total += s.Weight
		if best == nil || s.current > best.current {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	best.current -= total
	return best
}

// LeastConn is nginx's least_conn: lowest conns/weight wins, and servers
// that tie are taken in turn by weighted round robin.
func (p *Pool) LeastConn() *Server {
	p.mu.Lock()
	defer p.mu.Unlock()

	var tied []*Server
	for _, s := range p.servers {
		switch {
		case len(tied) == 0 || busier(tied[0], s):
			tied = []*Server{s}
		case !busier(s, tied[0]):
			tied = append(tied, s)
		}
	}
	return smoothPick(tied)
}

// RandomTwo is nginx's "random two least_conn": pick two servers at random
// (by weight), then keep the less busy one.
func (p *Pool) RandomTwo(rnd *rand.Rand) *Server {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch len(p.servers) {
	case 0:
		return nil
	case 1:
		return p.servers[0]
	}
	a := p.weightedPick(rnd, nil)
	b := p.weightedPick(rnd, a)
	if busier(a, b) {
		return b
	}
	return a
}

// weightedPick returns a random server in proportion to its weight,
// never the one in skip. Callers hold p.mu.
func (p *Pool) weightedPick(rnd *rand.Rand, skip *Server) *Server {
	total := 0
	for _, s := range p.servers {
		if s != skip {
			total += s.Weight
		}
	}
	n := rnd.IntN(total)
	for _, s := range p.servers {
		if s == skip {
			continue
		}
		if n < s.Weight {
			return s
		}
		n -= s.Weight
	}
	return nil // unreachable: n < total
}

// busier reports whether a carries more load per unit of weight than b.
// Cross-multiplying avoids fractions.
func busier(a, b *Server) bool {
	return a.conns*b.Weight > b.conns*a.Weight
}

// Acquire marks the start of a request on s, Release its end.
func (p *Pool) Acquire(s *Server) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s.conns++
}

func (p *Pool) Release(s *Server) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s.conns--
}

// IPHashKey mimics ip_hash: only the first three octets of an IPv4
// address are the key, so a whole /24 lands on one server.
func IPHashKey(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip // IPv6: nginx hashes the whole address
	}
	return strings.Join(parts[:3], ".")
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s)) // hash.Hash.Write never returns an error
	return h.Sum64()
}

// ModPick is plain "hash key; mod the server count", what hash without
// the consistent parameter amounts to.
func ModPick(key string, names []string) string {
	return names[hash64(key)%uint64(len(names))]
}

// Ring is a consistent hash ring, the idea behind "hash key consistent".
type Ring struct {
	points []point
}

type point struct {
	hash uint64
	name string
}

// NewRing puts vnodes points on the ring for every name. nginx documents
// its consistent method as ketama compatible, with 160 points.
func NewRing(names []string, vnodes int) *Ring {
	r := &Ring{}
	for _, name := range names {
		for i := 0; i < vnodes; i++ {
			r.points = append(r.points, point{hash64(name + "#" + strconv.Itoa(i)), name})
		}
	}
	sort.Slice(r.points, func(i, j int) bool { return r.points[i].hash < r.points[j].hash })
	return r
}

// Get returns the first point clockwise from the key.
func (r *Ring) Get(key string) string {
	h := hash64(key)
	i, _ := slices.BinarySearchFunc(r.points, h, func(p point, h uint64) int {
		switch {
		case p.hash < h:
			return -1
		case p.hash > h:
			return 1
		}
		return 0
	})
	if i == len(r.points) {
		i = 0
	}
	return r.points[i].name
}
