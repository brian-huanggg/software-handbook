package dbclock

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// CreateApp 用節點自己的時鐘算 expires_at，再當參數傳給資料庫。
func CreateApp(ctx context.Context, db *sql.DB, n Node, code string, ttl time.Duration) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO links (code, expires_at) VALUES ($1, $2)`,
		code, n.Now().Add(ttl))
	return err
}

// IsActiveApp 把 expires_at 讀回應用程式，再和節點自己的時鐘比。
func IsActiveApp(ctx context.Context, db *sql.DB, n Node, code string) (bool, error) {
	var exp time.Time
	err := db.QueryRowContext(ctx,
		`SELECT expires_at FROM links WHERE code = $1`, code).Scan(&exp)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	return exp.After(n.Now()), nil
}

// CreateDB 讓資料庫自己算：expires_at = now() + ttl，應用程式不傳任何「現在」。
func CreateDB(ctx context.Context, db *sql.DB, code string, ttl time.Duration) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO links (code, expires_at) VALUES ($1, now() + make_interval(secs => $2))`,
		code, ttl.Seconds())
	return err
}

// IsActiveDB 把比較留在 SQL 裡：expires_at > now()。
func IsActiveDB(ctx context.Context, db *sql.DB, code string) (bool, error) {
	var active bool
	err := db.QueryRowContext(ctx,
		`SELECT expires_at > now() FROM links WHERE code = $1`, code).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	return active, err
}

// Clocks 是同一個交易裡三個「現在」的讀數。
type Clocks struct {
	TxStart   time.Time // now()：交易開始的時間，交易內不變
	Statement time.Time // statement_timestamp()：這條語句開始的時間
	Wall      time.Time // clock_timestamp()：呼叫當下的真實時間
}

// ReadClocks 在同一個交易裡，每隔 gap 讀一次三個時間，共 n 次。
func ReadClocks(ctx context.Context, db *sql.DB, n int, gap time.Duration) ([]Clocks, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var out []Clocks
	for range n {
		var c Clocks
		if err := tx.QueryRowContext(ctx,
			`SELECT now(), statement_timestamp(), clock_timestamp()`).
			Scan(&c.TxStart, &c.Statement, &c.Wall); err != nil {
			return nil, err
		}
		out = append(out, c)
		time.Sleep(gap)
	}
	return out, tx.Commit()
}

// StoredAs 把 now() 同時存進 timestamp 與 timestamptz 欄位，再用指定的 session 時區讀回。
// 回傳兩欄的 epoch 秒，用來看「不帶時區」的欄位在不同時區下會差多少。
func StoredAs(ctx context.Context, db *sql.DB, writeTZ, readTZ string) (plain, tz float64, err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()

	if _, err = conn.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS t (a timestamp, b timestamptz)`); err != nil {
		return 0, 0, err
	}
	if _, err = conn.ExecContext(ctx, `TRUNCATE t`); err != nil {
		return 0, 0, err
	}
	if _, err = conn.ExecContext(ctx, `SET TIME ZONE '`+writeTZ+`'`); err != nil {
		return 0, 0, err
	}
	if _, err = conn.ExecContext(ctx, `INSERT INTO t VALUES (now(), now())`); err != nil {
		return 0, 0, err
	}
	if _, err = conn.ExecContext(ctx, `SET TIME ZONE '`+readTZ+`'`); err != nil {
		return 0, 0, err
	}
	// 把兩欄都當成 readTZ 的本地時間轉成 epoch，才能直接相減。
	err = conn.QueryRowContext(ctx,
		`SELECT extract(epoch FROM a AT TIME ZONE current_setting('TimeZone')), extract(epoch FROM b) FROM t`).
		Scan(&plain, &tz)
	return plain, tz, err
}
