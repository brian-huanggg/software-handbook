// demo 依序印出文章用到的實測結果。先執行 make up。
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	dblocks "handbook/playground/backend/database-locks"
)

func main() {
	ctx := context.Background()
	db, err := dblocks.Open(dblocks.DefaultDSN)
	if err != nil {
		log.Fatalf("資料庫未啟動（先執行 make up）：%v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(60)

	must(compat(ctx, db))
	must(quota(ctx, db))
	must(deadlock(ctx, db))
	must(skipLocked(ctx, db))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// compat 印出四種行鎖的相容矩陣。
func compat(ctx context.Context, db *sql.DB) error {
	if err := dblocks.Reset(ctx, db, 2, 10); err != nil {
		return err
	}
	fmt.Println("== 行鎖相容矩陣（X = 會被擋住）==")
	fmt.Printf("%-20s", "持有 \\ 請求")
	for _, m := range dblocks.Modes {
		fmt.Printf("%-20s", m)
	}
	fmt.Println()
	for _, h := range dblocks.Modes {
		fmt.Printf("%-20s", h)
		for _, r := range dblocks.Modes {
			c, err := dblocks.Conflicts(ctx, db, 1, h, r)
			if err != nil {
				return err
			}
			mark := "ok"
			if c {
				mark = "X"
			}
			fmt.Printf("%-20s", mark)
		}
		fmt.Println()
	}
	return nil
}

// quota：額度 30，50 個請求同時扣，比較四種寫法。
func quota(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n== 額度 30，50 個請求同時扣 ==")
	strategies := []struct {
		name string
		fn   func(context.Context, *sql.DB, int64) error
	}{
		{"沒有鎖（先讀再寫）", dblocks.ConsumeNaive},
		{"悲觀鎖（FOR UPDATE）", dblocks.ConsumeLocked},
		{"原子 UPDATE", dblocks.ConsumeAtomic},
		{"樂觀鎖（版本，重試 3 次）", func(ctx context.Context, db *sql.DB, id int64) error {
			return dblocks.ConsumeOptimistic(ctx, db, id, 3)
		}},
	}
	for _, s := range strategies {
		if err := dblocks.Reset(ctx, db, 1, 30); err != nil {
			return err
		}
		var ok, noQuota, conflict, other int
		var mu sync.Mutex
		var wg sync.WaitGroup
		start := time.Now()
		for range 50 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := s.fn(ctx, db, 1)
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					ok++
				case errors.Is(err, dblocks.ErrNoQuota):
					noQuota++
				case errors.Is(err, dblocks.ErrConflict):
					conflict++
				default:
					other++
				}
			}()
		}
		wg.Wait()
		left, err := dblocks.Quota(ctx, db, 1)
		if err != nil {
			return err
		}
		fmt.Printf("%-24s 成功 %2d，額度用完 %2d，衝突放棄 %2d，其他錯誤 %d；剩餘額度 %2d；耗時 %v\n",
			s.name, ok, noQuota, conflict, other, left, time.Since(start).Round(time.Millisecond))
	}
	return nil
}

// deadlock：兩個交易以相反順序更新兩列。
func deadlock(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n== deadlock：A 搬 1→2，B 搬 2→1 ==")
	run := func(name string, move func(context.Context, *sql.DB, int64, int64, int, time.Duration) error) error {
		if err := dblocks.Reset(ctx, db, 2, 100); err != nil {
			return err
		}
		errs := make([]error, 2)
		var wg sync.WaitGroup
		pairs := [][2]int64{{1, 2}, {2, 1}}
		start := time.Now()
		for i, p := range pairs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = move(ctx, db, p[0], p[1], 10, 100*time.Millisecond)
			}()
		}
		wg.Wait()
		for i, e := range errs {
			switch {
			case e == nil:
				fmt.Printf("%s：交易 %c 成功\n", name, 'A'+i)
			case dblocks.SQLState(e) == dblocks.CodeDeadlock:
				fmt.Printf("%s：交易 %c 被資料庫回滾（SQLSTATE %s）\n", name, 'A'+i, dblocks.CodeDeadlock)
			default:
				return e
			}
		}
		fmt.Printf("%s：耗時 %v\n", name, time.Since(start).Round(time.Millisecond))
		return nil
	}
	if err := run("相反順序", dblocks.MoveQuota); err != nil {
		return err
	}
	return run("固定順序", dblocks.MoveQuotaOrdered)
}

func skipLocked(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n== SKIP LOCKED：id 1 被鎖住時，另一個交易挑列 ==")
	if err := dblocks.Reset(ctx, db, 5, 10); err != nil {
		return err
	}
	ids, err := dblocks.SkipLocked(ctx, db)
	if err != nil {
		return err
	}
	fmt.Println("拿到的 id：", ids)
	return nil
}
