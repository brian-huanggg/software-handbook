// Package analytics is the starter code for the boundary exercise.
package analytics

import "sync"

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

func (r *Recorder) Total(code string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.totals[code]
}
