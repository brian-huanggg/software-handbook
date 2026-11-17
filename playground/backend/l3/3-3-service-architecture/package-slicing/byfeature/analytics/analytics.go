// Package analytics counts clicks and reports the top links, feature style.
package analytics

import (
	"sort"
	"sync"
)

type Count struct {
	Code  string
	Total int
}

// Recorder satisfies link.ClickRecorder without importing package link:
// Go interfaces are satisfied implicitly.
type Recorder struct {
	mu     sync.Mutex
	totals map[string]int
}

func New() *Recorder { return &Recorder{totals: map[string]int{}} }

func (r *Recorder) Record(code string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totals[code]++
}

func (r *Recorder) Top(n int) []Count {
	r.mu.Lock()
	out := make([]Count, 0, len(r.totals))
	for code, total := range r.totals {
		out = append(out, Count{Code: code, Total: total})
	}
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Code < out[j].Code
	})
	return out[:min(n, len(out))]
}
