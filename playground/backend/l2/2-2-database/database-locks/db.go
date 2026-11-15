// Package dblocks 在 PostgreSQL 上重現行鎖、悲觀鎖、樂觀鎖與 deadlock。
// 情境是短網址服務 v2 的「點擊額度」：每個短網址只能被點 quota 次。
package dblocks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DefaultDSN 對應 compose.yaml 的 port。
const DefaultDSN = "postgres://postgres:demo@127.0.0.1:55434/demo?sslmode=disable"

// PostgreSQL 的 SQLSTATE：
const (
	CodeDeadlock     = "40P01" // deadlock_detected
	CodeLockNotAvail = "55P03" // lock_not_available，NOWAIT 搶不到鎖時
)

var ErrNoQuota = errors.New("quota exhausted")

// Open 連上資料庫並確認連得到；連不上時回傳錯誤，呼叫端決定要略過還是失敗。
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Reset 重建 links 資料表，並放入 id 1 到 n 的短網址，每筆額度都是 quota。
func Reset(ctx context.Context, db *sql.DB, n, quota int) error {
	stmts := []string{
		`DROP TABLE IF EXISTS links`,
		`CREATE TABLE links (
			id      BIGINT PRIMARY KEY,
			code    TEXT   NOT NULL UNIQUE,
			quota   INT    NOT NULL CHECK (quota >= 0),
			clicks  INT    NOT NULL DEFAULT 0,
			version INT    NOT NULL DEFAULT 0
		)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	for i := 1; i <= n; i++ {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO links (id, code, quota) VALUES ($1, $2, $3)`,
			i, fmt.Sprintf("c%d", i), quota); err != nil {
			return err
		}
	}
	return nil
}

// Quota 讀出某筆短網址目前剩下的額度。
func Quota(ctx context.Context, db *sql.DB, id int64) (int, error) {
	var q int
	err := db.QueryRowContext(ctx, `SELECT quota FROM links WHERE id = $1`, id).Scan(&q)
	return q, err
}

// SQLState 取出 PostgreSQL 錯誤的 SQLSTATE；不是資料庫錯誤時回傳空字串。
func SQLState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
