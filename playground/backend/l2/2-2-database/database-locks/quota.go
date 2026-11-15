package dblocks

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// work 模擬讀完之後、寫回之前的處理時間（驗證網址、查黑名單等）。
// 時間窗越長，競態越容易出現；真實服務裡它只會更長。
const work = 5 * time.Millisecond

// ConsumeNaive 先讀再算再寫，沒有任何鎖。
func ConsumeNaive(ctx context.Context, db *sql.DB, id int64) error {
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
		return ErrNoQuota
	}
	time.Sleep(work)
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = $1, clicks = clicks + 1 WHERE id = $2`, q-1, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ConsumeLocked 悲觀鎖：讀的時候就用 FOR UPDATE 鎖住那一列，
// 其他想鎖同一列的交易要等到這個交易結束。
func ConsumeLocked(ctx context.Context, db *sql.DB, id int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var q int
	if err := tx.QueryRowContext(ctx, `SELECT quota FROM links WHERE id = $1 FOR UPDATE`, id).Scan(&q); err != nil {
		return err
	}
	if q <= 0 {
		return ErrNoQuota
	}
	time.Sleep(work)
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = $1, clicks = clicks + 1 WHERE id = $2`, q-1, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ConsumeAtomic 把判斷與扣減合成一個語句，不需要先讀。
func ConsumeAtomic(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx,
		`UPDATE links SET quota = quota - 1, clicks = clicks + 1 WHERE id = $1 AND quota > 0`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoQuota
	}
	return nil
}

// ErrConflict 表示樂觀鎖重試用完仍然衝突。
var ErrConflict = errors.New("version conflict, retries exhausted")

// ConsumeOptimistic 樂觀鎖：不加鎖地讀，寫的時候檢查版本沒變。
// 版本被別人改過就重讀重試。細節留給下一篇「版本欄位與條件更新」。
func ConsumeOptimistic(ctx context.Context, db *sql.DB, id int64, maxRetries int) error {
	for range maxRetries + 1 {
		var q, ver int
		if err := db.QueryRowContext(ctx,
			`SELECT quota, version FROM links WHERE id = $1`, id).Scan(&q, &ver); err != nil {
			return err
		}
		if q <= 0 {
			return ErrNoQuota
		}
		time.Sleep(work)
		res, err := db.ExecContext(ctx,
			`UPDATE links SET quota = $1, clicks = clicks + 1, version = version + 1
			 WHERE id = $2 AND version = $3`, q-1, id, ver)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 1 {
			return nil
		}
	}
	return ErrConflict
}
