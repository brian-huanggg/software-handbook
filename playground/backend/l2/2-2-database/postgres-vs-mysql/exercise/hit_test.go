//go:build exercise

package exercise

import (
	"context"
	"database/sql"
	"sort"
	"sync"
	"testing"

	pgvsmysql "handbook/playground/backend/postgres-vs-mysql"
)

func TestHit(t *testing.T) {
	tests := []struct {
		d   pgvsmysql.Dialect
		dsn string
	}{
		{pgvsmysql.Postgres, pgvsmysql.DefaultPostgresDSN},
		{pgvsmysql.MySQL, pgvsmysql.DefaultMySQLDSN},
	}
	for _, tc := range tests {
		t.Run(tc.d.String(), func(t *testing.T) {
			db, err := pgvsmysql.Open(tc.d, tc.dsn)
			if err != nil {
				t.Fatalf("資料庫未啟動（先執行 make up）：%v", err)
			}
			defer db.Close()
			db.SetMaxOpenConns(20)

			ctx := context.Background()
			for _, s := range []string{
				`DROP TABLE IF EXISTS hits`,
				`CREATE TABLE hits (code VARCHAR(16) PRIMARY KEY, clicks BIGINT NOT NULL)`,
			} {
				if _, err := db.ExecContext(ctx, s); err != nil {
					t.Fatal(err)
				}
			}

			const n = 50
			got := make([]int64, n)
			var wg sync.WaitGroup
			errs := make(chan error, n)
			for i := range n {
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, err := Hit(ctx, db, tc.d, "abc")
					if err != nil {
						errs <- err
						return
					}
					got[i] = v
				}()
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				t.Fatal(err)
			}

			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			for i, v := range got {
				if v != int64(i+1) {
					t.Fatalf("排序後第 %d 個回傳值是 %d，應該是 %d（回傳值必須各不相同）", i, v, i+1)
				}
			}
			assertTotal(t, db, n)
		})
	}
}

func assertTotal(t *testing.T, db *sql.DB, want int64) {
	t.Helper()
	var total int64
	if err := db.QueryRow(`SELECT clicks FROM hits WHERE code = 'abc'`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != want {
		t.Fatalf("clicks = %d，應該是 %d", total, want)
	}
}
