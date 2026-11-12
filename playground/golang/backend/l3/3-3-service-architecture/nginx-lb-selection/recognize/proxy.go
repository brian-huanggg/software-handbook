//go:build recognize

// Package recognize is AI-generated code for the review exercise.
package recognize

import (
	"hash/fnv"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

// Proxy sends each user to one backend so that the in-memory session
// stays valid. A health checker calls Remove when a backend stops
// answering.
type Proxy struct {
	mu       sync.RWMutex
	backends []*url.URL
}

func NewProxy(backends ...*url.URL) *Proxy {
	return &Proxy{backends: backends}
}

func (p *Proxy) pick(r *http.Request) *url.URL {
	h := fnv.New32a()
	h.Write([]byte(r.RemoteAddr))

	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.backends) == 0 {
		return nil
	}
	return p.backends[h.Sum32()%uint32(len(p.backends))]
}

// Remove drops a backend that failed its health check.
func (p *Proxy) Remove(dead *url.URL) {
	p.mu.Lock()
	defer p.mu.Unlock()
	kept := p.backends[:0]
	for _, b := range p.backends {
		if b.String() != dead.String() {
			kept = append(kept, b)
		}
	}
	p.backends = kept
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := p.pick(r)
	if target == nil {
		http.Error(w, "no backend available", http.StatusServiceUnavailable)
		return
	}
	httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
}
