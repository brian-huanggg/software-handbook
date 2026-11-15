//go:build exercise

package exercise

import (
	"context"
	"errors"
	"sync"
	"testing"

	dblocks "handbook/playground/backend/database-locks"
)

func TestConsume(t *testing.T) {
	ctx := context.Background()
	db, err := dblocks.Open(dblocks.DefaultDSN)
	if err != nil {
		t.Fatalf("資料庫未啟動（先執行 make up）：%v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(60)
	if err := dblocks.Reset(ctx, db, 1, 30); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	ok := 0
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := Consume(ctx, db, 1)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				ok++
			} else if !errors.Is(err, dblocks.ErrNoQuota) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	left, err := dblocks.Quota(ctx, db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok != 30 || left != 0 {
		t.Fatalf("成功 %d 次、剩餘額度 %d，預期 30 次與 0", ok, left)
	}
}

func TestTransfer(t *testing.T) {
	ctx := context.Background()
	db, err := dblocks.Open(dblocks.DefaultDSN)
	if err != nil {
		t.Fatalf("資料庫未啟動（先執行 make up）：%v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(60)
	if err := dblocks.Reset(ctx, db, 2, 1000); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := range 20 {
		from, to := int64(1), int64(2)
		if i%2 == 1 {
			from, to = to, from
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Transfer(ctx, db, from, to, 1); err != nil {
				if dblocks.SQLState(err) == dblocks.CodeDeadlock {
					t.Errorf("%d→%d 遇到 deadlock", from, to)
					return
				}
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	a, err := dblocks.Quota(ctx, db, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := dblocks.Quota(ctx, db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if a+b != 2000 {
		t.Fatalf("額度總和 %d，預期 2000", a+b)
	}
}
