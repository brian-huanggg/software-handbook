// Package link creates short links and resolves them, feature style.
package link

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("link not found")

type Link struct {
	Code string
	URL  string
}

// ClickRecorder is defined here, by the code that needs it, so that
// link never imports analytics. main wires in the real implementation.
type ClickRecorder interface {
	Record(code string)
}

// store is unexported: nothing outside this package can touch the map.
type store struct {
	mu    sync.RWMutex
	links map[string]Link
}

type Service struct {
	links  store
	clicks ClickRecorder
}

func New(clicks ClickRecorder) *Service {
	return &Service{links: store{links: map[string]Link{}}, clicks: clicks}
}

func (s *Service) Create(code, url string) Link {
	s.links.mu.Lock()
	defer s.links.mu.Unlock()
	l := Link{Code: code, URL: url}
	s.links.links[code] = l
	return l
}

func (s *Service) Resolve(code string) (string, error) {
	s.links.mu.RLock()
	l, ok := s.links.links[code]
	s.links.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	s.clicks.Record(code)
	return l.URL, nil
}
