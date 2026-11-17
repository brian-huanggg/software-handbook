// Package link is the starter code for the boundary exercise.
package link

import (
	"errors"
	"sync"

	"handbook/playground/backend/package-slicing/exercise/analytics"
)

var ErrNotFound = errors.New("link not found")

type Store struct {
	Mu    sync.RWMutex
	Links map[string]string
}

type Service struct {
	Store  *Store
	Clicks *analytics.Recorder
}

func New(clicks *analytics.Recorder) *Service {
	return &Service{Store: &Store{Links: map[string]string{}}, Clicks: clicks}
}

func (s *Service) Create(code, url string) {
	s.Store.Mu.Lock()
	defer s.Store.Mu.Unlock()
	s.Store.Links[code] = url
}

func (s *Service) Resolve(code string) (string, error) {
	s.Store.Mu.RLock()
	url, ok := s.Store.Links[code]
	s.Store.Mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	s.Clicks.Record(code)
	return url, nil
}
