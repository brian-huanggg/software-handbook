//go:build exercise

package exercise

import (
	"context"
	"database/sql"
	"time"

	dblocks "handbook/playground/backend/database-locks"
)

// Consume 扣掉某筆短網址的 1 點額度；額度已經是 0 就回傳 dblocks.ErrNoQuota。
// 目前的寫法在高並行下會讓額度被超扣，請修好它。
func Consume(ctx context.Context, db *sql.DB, id int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var q int
	if err := tx.QueryRowContext(ctx, `SELECT quota FROM links WHERE id = $1`, id).Scan(&q); err != nil {
		return err
	}
	if q <= 0 {
		return dblocks.ErrNoQuota
	}
	time.Sleep(5 * time.Millisecond) // 模擬驗證網址的處理時間
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = $1 WHERE id = $2`, q-1, id); err != nil {
		return err
	}
	return tx.Commit()
}
