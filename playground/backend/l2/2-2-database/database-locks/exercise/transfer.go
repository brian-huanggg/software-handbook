//go:build exercise

package exercise

import (
	"context"
	"database/sql"
	"time"
)

// Transfer 把 n 點額度從 from 搬到 to，兩個更新在同一個交易裡。
// 目前兩個方向同時搬時，資料庫會偵測到 deadlock 並回滾其中一個，請修好它。
func Transfer(ctx context.Context, db *sql.DB, from, to int64, n int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = quota - $1 WHERE id = $2`, n, from); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond) // 模擬兩個更新之間的處理時間
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = quota + $1 WHERE id = $2`, n, to); err != nil {
		return err
	}
	return tx.Commit()
}
