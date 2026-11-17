//go:build recognize

// Package analytics reports the most clicked links. AI-generated for the
// review exercise; it is not compiled.
package analytics

import (
	"sort"

	"handbook/playground/backend/package-slicing/recognize/common"
	"handbook/playground/backend/package-slicing/recognize/link"
)

type Report struct {
	Links  *link.Store
	Clicks *ClickStore
}

func (r *Report) Top(n int) []common.LinkStat {
	totals := map[string]int{}
	for _, c := range r.Clicks.All() {
		totals[c.Code]++
	}
	out := make([]common.LinkStat, 0, len(totals))
	for code, total := range totals {
		l, ok := r.Links.Rows[code]
		if !ok {
			continue
		}
		out = append(out, common.LinkStat{
			Code:  code,
			URL:   common.NormalizeURL(l.URL),
			Total: total,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out[:min(n, len(out))]
}
