//go:build exercise

package exercise

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dbclock "handbook/playground/backend/db-clock"
)

// LocalSkew 模擬「這台機器的時鐘比真實時間快 90 秒」。
const LocalSkew = 90 * time.Second

func localNow() time.Time { return time.Now().Add(LocalSkew) }

// Renew 為還沒過期的短網址續期：新的 expires_at 是「現在 + ttl」。
// 已過期的短網址回傳 dbclock.ErrExpired，不存在的回傳 dbclock.ErrNotFound。
func Renew(ctx context.Context, db *sql.DB, code string, ttl time.Duration) error {
	var exp time.Time
	err := db.QueryRowContext(ctx, `SELECT expires_at FROM links WHERE code = $1`, code).Scan(&exp)
	if errors.Is(err, sql.ErrNoRows) {
		return dbclock.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !exp.After(localNow()) {
		return dbclock.ErrExpired
	}
	_, err = db.ExecContext(ctx,
		`UPDATE links SET expires_at = $1 WHERE code = $2`, localNow().Add(ttl), code)
	return err
}
