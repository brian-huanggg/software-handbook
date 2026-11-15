package dblocks

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"
)

func testDB(t *testing.T) (context.Context, *sql.DB) {
	t.Helper()
	db, err := Open(DefaultDSN)
	if err != nil {
		t.Skipf("資料庫未啟動（先執行 make up）：%v", err)
	}
	db.SetMaxOpenConns(60)
	t.Cleanup(func() { db.Close() })
	return context.Background(), db
}

func TestConflictsMatrix(t *testing.T) {
	ctx, h := testDB(t)
	if err := Reset(ctx, h, 2, 10); err != nil {
		t.Fatal(err)
	}
	// 官方文件的行鎖衝突表：true 代表衝突。
	want := map[[2]string]bool{
		{"FOR KEY SHARE", "FOR UPDATE"}:        true,
		{"FOR KEY SHARE", "FOR NO KEY UPDATE"}: false,
		{"FOR SHARE", "FOR NO KEY UPDATE"}:     true,
		{"FOR SHARE", "FOR SHARE"}:             false,
		{"FOR NO KEY UPDATE", "FOR KEY SHARE"}: false,
		{"FOR NO KEY UPDATE", "FOR SHARE"}:     true,
		{"FOR UPDATE", "FOR KEY SHARE"}:        true,
		{"FOR UPDATE", "FOR UPDATE"}:           true,
	}
	for k, w := range want {
		got, err := Conflicts(ctx, h, 1, k[0], k[1])
		if err != nil {
			t.Fatal(err)
		}
		if got != w {
			t.Errorf("持有 %s、請求 %s：衝突=%v，預期 %v", k[0], k[1], got, w)
		}
	}
}

func consumeAll(t *testing.T, n int, fn func(context.Context, *sql.DB) error) (ok int) {
	t.Helper()
	ctx, h := testDB(t)
	if err := Reset(ctx, h, 1, 30); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := fn(ctx, h)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				ok++
			} else if !errors.Is(err, ErrNoQuota) && !errors.Is(err, ErrConflict) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	return ok
}

func TestNaiveOversells(t *testing.T) {
	ok := consumeAll(t, 50, func(ctx context.Context, h *sql.DB) error { return ConsumeNaive(ctx, h, 1) })
	if ok <= 30 {
		t.Errorf("沒有鎖時成功 %d 次，預期超過額度 30（競態沒有重現，可重跑）", ok)
	}
}

func TestLockedExact(t *testing.T) {
	ok := consumeAll(t, 50, func(ctx context.Context, h *sql.DB) error { return ConsumeLocked(ctx, h, 1) })
	if ok != 30 {
		t.Errorf("成功 %d 次，預期正好 30", ok)
	}
}

func TestAtomicExact(t *testing.T) {
	ok := consumeAll(t, 50, func(ctx context.Context, h *sql.DB) error { return ConsumeAtomic(ctx, h, 1) })
	if ok != 30 {
		t.Errorf("成功 %d 次，預期正好 30", ok)
	}
}

func TestOptimisticNeverOversells(t *testing.T) {
	ok := consumeAll(t, 50, func(ctx context.Context, h *sql.DB) error { return ConsumeOptimistic(ctx, h, 1, 3) })
	if ok > 30 {
		t.Errorf("樂觀鎖成功 %d 次，不應超過 30", ok)
	}
}

func TestDeadlock(t *testing.T) {
	ctx, h := testDB(t)
	run := func(move func(context.Context, *sql.DB, int64, int64) error) (deadlocks int) {
		if err := Reset(ctx, h, 2, 100); err != nil {
			t.Fatal(err)
		}
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i, p := range [][2]int64{{1, 2}, {2, 1}} {
			wg.Add(1)
			go func() { defer wg.Done(); errs[i] = move(ctx, h, p[0], p[1]) }()
		}
		wg.Wait()
		for _, e := range errs {
			if SQLState(e) == CodeDeadlock {
				deadlocks++
			} else if e != nil {
				t.Fatal(e)
			}
		}
		return deadlocks
	}
	if n := run(func(ctx context.Context, h *sql.DB, a, b int64) error {
		return MoveQuota(ctx, h, a, b, 10, 100*time.Millisecond)
	}); n != 1 {
		t.Errorf("相反順序：deadlock 次數 %d，預期 1", n)
	}
	if n := run(func(ctx context.Context, h *sql.DB, a, b int64) error {
		return MoveQuotaOrdered(ctx, h, a, b, 10, 100*time.Millisecond)
	}); n != 0 {
		t.Errorf("固定順序：deadlock 次數 %d，預期 0", n)
	}
}

func TestSkipLocked(t *testing.T) {
	ctx, h := testDB(t)
	if err := Reset(ctx, h, 5, 10); err != nil {
		t.Fatal(err)
	}
	ids, err := SkipLocked(ctx, h)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
		t.Errorf("拿到 %v，預期 [2 3]", ids)
	}
}
