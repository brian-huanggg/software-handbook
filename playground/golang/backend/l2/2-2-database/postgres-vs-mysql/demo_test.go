package pgvsmysql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
)

// 這些測試需要 make up 起來的資料庫；連不上就略過，不讓 go test ./... 失敗。
func open(t *testing.T, d Dialect) *sql.DB {
	t.Helper()
	dsn, env := DefaultPostgresDSN, "PG_DSN"
	if d == MySQL {
		dsn, env = DefaultMySQLDSN, "MYSQL_DSN"
	}
	if v := os.Getenv(env); v != "" {
		dsn = v
	}
	db, err := Open(d, dsn)
	if err != nil {
		t.Skipf("資料庫未啟動（先執行 make up）：%v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDefaultIsolation(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		d    Dialect
		want string
	}{
		{Postgres, "read committed"},
		{MySQL, "REPEATABLE-READ"},
	}
	for _, tc := range tests {
		t.Run(tc.d.String(), func(t *testing.T) {
			got, err := DefaultIsolation(ctx, open(t, tc.d), tc.d)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("隔離級別 = %q，想要 %q", got, tc.want)
			}
		})
	}
}

func TestNonRepeatableRead(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		d           Dialect
		first, next int
	}{
		{Postgres, 1, 2}, // Read Committed：每個語句看到最新已提交的資料
		{MySQL, 1, 1},    // Repeatable Read：整個交易沿用第一次讀取的快照
	}
	for _, tc := range tests {
		t.Run(tc.d.String(), func(t *testing.T) {
			first, next, err := NonRepeatableRead(ctx, open(t, tc.d), tc.d)
			if err != nil {
				t.Fatal(err)
			}
			if first != tc.first || next != tc.next {
				t.Fatalf("兩次筆數 = %d → %d，想要 %d → %d", first, next, tc.first, tc.next)
			}
		})
	}
}

func TestGapInsert(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		d           Dialect
		wantBlocked bool
	}{
		{Postgres, false},
		{MySQL, true},
	}
	for _, tc := range tests {
		t.Run(tc.d.String(), func(t *testing.T) {
			blocked, err := GapInsert(ctx, open(t, tc.d), tc.d, 1500*time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			if blocked != tc.wantBlocked {
				t.Fatalf("blocked = %v，想要 %v", blocked, tc.wantBlocked)
			}
		})
	}
}

func TestSpaceAfterUpdates(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		d    Dialect
		grow bool // 全表 UPDATE 兩次後，總大小是否超過載入後的 1.5 倍
	}{
		{Postgres, true},
		{MySQL, false},
	}
	for _, tc := range tests {
		t.Run(tc.d.String(), func(t *testing.T) {
			stages, err := SpaceAfterUpdates(ctx, open(t, tc.d), tc.d, 50_000, 2)
			if err != nil {
				t.Fatal(err)
			}
			loaded, afterTwo := stages[0].Bytes, stages[2].Bytes
			if got := float64(afterTwo) > 1.5*float64(loaded); got != tc.grow {
				t.Fatalf("載入後 %d，更新兩次後 %d；是否超過 1.5 倍 = %v，想要 %v", loaded, afterTwo, got, tc.grow)
			}
			if tc.d == Postgres {
				// VACUUM 把空間標成可重用，但不還給作業系統：大小不會變小。
				if after := stages[3].Bytes; after < afterTwo*9/10 {
					t.Fatalf("VACUUM 後 %d 比之前 %d 小很多", after, afterTwo)
				}
			}
		})
	}
}
