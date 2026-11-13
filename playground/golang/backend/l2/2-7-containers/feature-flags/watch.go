package flags

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Watch 定期讀取檔案並重新載入。讀檔或驗證失敗時只記錄，繼續使用上一份設定。
func (s *Store) Watch(ctx context.Context, path string, every time.Duration) {
	var lastMod time.Time
	reload := func() {
		info, err := os.Stat(path)
		if err != nil {
			slog.Error("flags: stat failed, keep last good config", "err", err)
			return
		}
		if info.ModTime().Equal(lastMod) {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Error("flags: read failed, keep last good config", "err", err)
			return
		}
		if err := s.Load(data); err != nil {
			slog.Error("flags: invalid config, keep last good config", "err", err)
			return
		}
		lastMod = info.ModTime()
		slog.Info("flags: reloaded", "path", path)
	}

	reload()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			reload()
		}
	}
}
