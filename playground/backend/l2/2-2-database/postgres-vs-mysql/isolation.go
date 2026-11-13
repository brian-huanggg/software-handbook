package pgvsmysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// DefaultIsolation 回傳連線預設的隔離級別。
func DefaultIsolation(ctx context.Context, db *sql.DB, d Dialect) (string, error) {
	q := `SHOW transaction_isolation`
	if d == MySQL {
		q = `SELECT @@transaction_isolation`
	}
	var level string
	if err := db.QueryRowContext(ctx, q).Scan(&level); err != nil {
		return "", err
	}
	return level, nil
}

// NonRepeatableRead 在預設隔離級別的交易裡數兩次 links 的筆數，
// 兩次之間由另一條連線新增一筆並提交。
func NonRepeatableRead(ctx context.Context, db *sql.DB, d Dialect) (first, second int, err error) {
	if err := ResetLinks(ctx, db, d); err != nil {
		return 0, 0, err
	}
	if _, err := db.ExecContext(ctx, d.Rebind(`INSERT INTO links (id, code) VALUES (?, ?)`), 1, "a"); err != nil {
		return 0, 0, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM links`).Scan(&first); err != nil {
		return 0, 0, err
	}
	// 交易外的另一條連線，autocommit 立刻提交。
	if _, err := db.ExecContext(ctx, d.Rebind(`INSERT INTO links (id, code) VALUES (?, ?)`), 2, "b"); err != nil {
		return 0, 0, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM links`).Scan(&second); err != nil {
		return 0, 0, err
	}
	return first, second, nil
}

// GapInsert 讓交易 A 鎖定一個沒有任何資料列的範圍，再讓交易 B 插入範圍內的 id。
// 回傳 B 是否在 wait 內被擋住。
func GapInsert(ctx context.Context, db *sql.DB, d Dialect, wait time.Duration) (blocked bool, err error) {
	if err := ResetLinks(ctx, db, d); err != nil {
		return false, err
	}
	for _, id := range []int{10, 30} {
		if _, err := db.ExecContext(ctx, d.Rebind(`INSERT INTO links (id, code) VALUES (?, ?)`), id, fmt.Sprint("c", id)); err != nil {
			return false, err
		}
	}

	a, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer a.Rollback()

	// 15 到 25 之間沒有資料，鎖不到任何列；有沒有東西被鎖，取決於資料庫怎麼定義「鎖住這個範圍」。
	rows, err := a.QueryContext(ctx, d.Rebind(`SELECT id FROM links WHERE id BETWEEN ? AND ? FOR UPDATE`), 15, 25)
	if err != nil {
		return false, err
	}
	rows.Close()

	bctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	b, err := db.BeginTx(bctx, nil)
	if err != nil {
		return false, err
	}
	defer b.Rollback()

	_, err = b.ExecContext(bctx, d.Rebind(`INSERT INTO links (id, code) VALUES (?, ?)`), 20, "new")
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, context.DeadlineExceeded):
		return true, nil
	default:
		return false, err
	}
}
