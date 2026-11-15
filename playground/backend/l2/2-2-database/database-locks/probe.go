package dblocks

import (
	"context"
	"database/sql"
)

// Modes 是 SELECT 可以加的四種行鎖，由弱到強。
var Modes = []string{"FOR KEY SHARE", "FOR SHARE", "FOR NO KEY UPDATE", "FOR UPDATE"}

// Conflicts 回答：有人已經用 holder 模式鎖住 id 那一列，
// 另一個交易想用 requester 模式鎖同一列，會被擋住嗎？
// 請求方加上 NOWAIT：搶不到就立刻回錯誤（55P03），不會卡住。
func Conflicts(ctx context.Context, db *sql.DB, id int64, holder, requester string) (bool, error) {
	a, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer a.Rollback()
	var x int64
	if err := a.QueryRowContext(ctx, `SELECT id FROM links WHERE id = $1 `+holder, id).Scan(&x); err != nil {
		return false, err
	}

	b, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer b.Rollback()
	err = b.QueryRowContext(ctx, `SELECT id FROM links WHERE id = $1 `+requester+` NOWAIT`, id).Scan(&x)
	if SQLState(err) == CodeLockNotAvail {
		return true, nil
	}
	return false, err
}

// SkipLocked 示範 SKIP LOCKED：一個交易鎖住 id 1，
// 另一個交易挑「沒被鎖的列」，只會拿到其他列。回傳第二個交易拿到的 id。
func SkipLocked(ctx context.Context, db *sql.DB) ([]int64, error) {
	a, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer a.Rollback()
	if _, err := a.ExecContext(ctx, `SELECT id FROM links WHERE id = 1 FOR UPDATE`); err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SELECT id FROM links ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
