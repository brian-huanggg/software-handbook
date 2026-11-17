// Package service holds every use case, layered style.
package service

import (
	"errors"
	"sort"
	"time"

	"handbook/playground/backend/package-slicing/bylayer/model"
	"handbook/playground/backend/package-slicing/bylayer/store"
)

var ErrNotFound = errors.New("link not found")

type LinkService struct {
	Links  *store.LinkStore
	Clicks *store.ClickStore
}

func (s *LinkService) Create(code, url string) model.Link {
	l := model.Link{Code: code, URL: url}
	s.Links.Save(l)
	return l
}

func (s *LinkService) Resolve(code string) (string, error) {
	l, ok := s.Links.Get(code)
	if !ok {
		return "", ErrNotFound
	}
	s.Clicks.Add(model.Click{Code: code, At: time.Now()})
	return l.URL, nil
}

type StatsService struct {
	Clicks *store.ClickStore
}

func (s *StatsService) Top(n int) []model.Count {
	totals := map[string]int{}
	for _, c := range s.Clicks.All() {
		totals[c.Code]++
	}
	out := make([]model.Count, 0, len(totals))
	for code, total := range totals {
		out = append(out, model.Count{Code: code, Total: total})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Code < out[j].Code
	})
	return out[:min(n, len(out))]
}
