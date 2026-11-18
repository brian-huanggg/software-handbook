package main

import (
	"sort"
	"strconv"
	"sync"
)

// Link is the one resource this API exposes.
type Link struct {
	Code     string `json:"code"`
	Target   string `json:"target"`
	Owner    string `json:"owner,omitempty"`
	Disabled bool   `json:"disabled"`
}

// Store keeps links in memory. Every request runs in its own goroutine,
// so the map needs a lock.
type Store struct {
	mu    sync.RWMutex
	links map[string]Link
	next  int64
}

func NewStore() *Store {
	return &Store{links: make(map[string]Link)}
}

func (s *Store) Get(code string) (Link, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.links[code]
	return l, ok
}

// Add stores target under a new server-chosen code. The counter only grows,
// so a code is never handed out twice, even after a delete.
func (s *Store) Add(target, owner string) Link {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		s.next++
		code := "g" + strconv.FormatInt(s.next, 36)
		if _, taken := s.links[code]; !taken {
			l := Link{Code: code, Target: target, Owner: owner}
			s.links[code] = l
			return l
		}
	}
}

// Put replaces the whole link at code and reports whether it is new.
func (s *Store) Put(code, target, owner string) (l Link, created bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.links[code]
	l = Link{Code: code, Target: target, Owner: owner}
	s.links[code] = l
	return l, !existed
}

func (s *Store) Delete(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.links[code]
	delete(s.links, code)
	return ok
}

// Disable marks a link as disabled. Calling it again changes nothing.
func (s *Store) Disable(code string) (Link, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.links[code]
	if !ok {
		return Link{}, false
	}
	l.Disabled = true
	s.links[code] = l
	return l, true
}

// List returns the links accepted by keep, ordered by code, after skipping
// offset matches and stopping at limit. It never returns nil.
func (s *Store) List(keep func(Link) bool, limit, offset int) []Link {
	s.mu.RLock()
	all := make([]Link, 0, len(s.links))
	for _, l := range s.links {
		if keep(l) {
			all = append(all, l)
		}
	}
	s.mu.RUnlock()

	sort.Slice(all, func(i, j int) bool { return all[i].Code < all[j].Code })
	if offset >= len(all) {
		return []Link{}
	}
	all = all[offset:]
	if len(all) > limit {
		all = all[:limit]
	}
	return all
}
