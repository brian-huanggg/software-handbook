package main

import (
	"context"
	"log/slog"
	"time"
)

// purger deletes expired rows and reports how many it removed.
type purger struct {
	name  string
	purge func(context.Context) (int64, error)
}

// runJanitor runs every purger once at start and then every interval, until
// ctx is done. Sessions and refresh tokens are only removed lazily when their
// credential comes back; without this, abandoned rows pile up forever.
func runJanitor(ctx context.Context, interval time.Duration, purgers ...purger) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		for _, p := range purgers {
			n, err := p.purge(ctx)
			switch {
			case err != nil && ctx.Err() == nil:
				slog.ErrorContext(ctx, "purge expired", "what", p.name, "err", err)
			case n > 0:
				slog.InfoContext(ctx, "purged expired", "what", p.name, "rows", n)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
