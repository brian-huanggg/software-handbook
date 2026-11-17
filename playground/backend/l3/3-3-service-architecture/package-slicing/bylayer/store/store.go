// Package store holds every repository, layered style.
package store

import (
	"sync"

	"handbook/playground/backend/package-slicing/bylayer/model"
)

// LinkStore must be exported: package service, one directory away, calls it.
type LinkStore struct {
	mu    sync.RWMutex
	links map[string]model.Link
}

func NewLinkStore() *LinkStore { return &LinkStore{links: map[string]model.Link{}} }

func (s *LinkStore) Save(l model.Link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links[l.Code] = l
}

func (s *LinkStore) Get(code string) (model.Link, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.links[code]
	return l, ok
}

type ClickStore struct {
	mu     sync.Mutex
	clicks []model.Click
}

func NewClickStore() *ClickStore { return &ClickStore{} }

func (s *ClickStore) Add(c model.Click) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clicks = append(s.clicks, c)
}

func (s *ClickStore) All() []model.Click {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.Click(nil), s.clicks...)
}
