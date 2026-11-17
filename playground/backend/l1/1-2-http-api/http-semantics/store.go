package main

import (
	"strconv"
	"sync"
)

// Store keeps short codes in memory. Every request runs in its own
// goroutine, so the map needs a lock.
type Store struct {
	mu    sync.RWMutex
	links map[string]string
	next  int64
}

func NewStore() *Store {
	return &Store{links: make(map[string]string)}
}

func (s *Store) Get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.links[code]
	return target, ok
}

// Put sets code to target and reports whether the code is new.
// Calling it twice with the same arguments leaves the same state: idempotent.
func (s *Store) Put(code, target string) (created bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.links[code]
	s.links[code] = target
	return !existed
}

// Add stores target under a new server-chosen code. Every call creates
// another link, which is why the POST that uses it is not idempotent.
func (s *Store) Add(target string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		s.next++
		code := "g" + strconv.FormatInt(s.next, 36)
		if _, taken := s.links[code]; !taken {
			s.links[code] = target
			return code
		}
	}
}

// Delete removes code and reports whether it existed.
func (s *Store) Delete(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.links[code]
	delete(s.links, code)
	return ok
}
