package dblocks

import (
	"context"
	"database/sql"
	"sort"
	"time"
)

// MoveQuota 把 n 點額度從 from 搬到 to，在同一個交易裡先扣再加。
// pause 是兩個 UPDATE 之間的間隔，用來放大 deadlock 的時間窗。
func MoveQuota(ctx context.Context, db *sql.DB, from, to int64, n int, pause time.Duration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = quota - $1 WHERE id = $2`, n, from); err != nil {
		return err
	}
	time.Sleep(pause)
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = quota + $1 WHERE id = $2`, n, to); err != nil {
		return err
	}
	return tx.Commit()
}

// MoveQuotaOrdered 做的事和 MoveQuota 一樣，但永遠依 id 由小到大加鎖。
func MoveQuotaOrdered(ctx context.Context, db *sql.DB, from, to int64, n int, pause time.Duration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ids := []int64{from, to}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		var q int
		if err := tx.QueryRowContext(ctx, `SELECT quota FROM links WHERE id = $1 FOR UPDATE`, id).Scan(&q); err != nil {
			return err
		}
		time.Sleep(pause)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = quota - $1 WHERE id = $2`, n, from); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = quota + $1 WHERE id = $2`, n, to); err != nil {
		return err
	}
	return tx.Commit()
}
