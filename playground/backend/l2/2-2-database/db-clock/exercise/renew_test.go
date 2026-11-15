//go:build exercise

package exercise

import (
	"context"
	"errors"
	"testing"
	"time"

	dbclock "handbook/playground/backend/db-clock"
)

func TestRenew(t *testing.T) {
	ctx := context.Background()
	db, err := dbclock.Open(dbclock.DefaultDSN)
	if err != nil {
		t.Fatalf("資料庫未啟動（先執行 make up）：%v", err)
	}
	defer db.Close()
	if err := dbclock.Reset(ctx, db); err != nil {
		t.Fatal(err)
	}

	// 還剩 60 秒：以資料庫的時間來看仍有效，必須能續期。
	if err := dbclock.CreateDB(ctx, db, "live", 60*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := dbclock.CreateDB(ctx, db, "gone", -time.Second); err != nil {
		t.Fatal(err)
	}

	if err := Renew(ctx, db, "live", time.Hour); err != nil {
		t.Fatalf("還有效的短網址應能續期：%v", err)
	}
	// 新的 expires_at 必須是「資料庫的現在 + 1 小時」，誤差 2 秒內。
	var off float64
	if err := db.QueryRowContext(ctx,
		`SELECT extract(epoch FROM expires_at - (now() + interval '1 hour')) FROM links WHERE code = 'live'`).
		Scan(&off); err != nil {
		t.Fatal(err)
	}
	if off < -2 || off > 2 {
		t.Errorf("expires_at 和資料庫時間差了 %.0f 秒", off)
	}

	if err := Renew(ctx, db, "gone", time.Hour); !errors.Is(err, dbclock.ErrExpired) {
		t.Errorf("已過期應回 ErrExpired，實際 %v", err)
	}
	if err := Renew(ctx, db, "nope", time.Hour); !errors.Is(err, dbclock.ErrNotFound) {
		t.Errorf("不存在應回 ErrNotFound，實際 %v", err)
	}
}
