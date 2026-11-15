// Package dbclock 在 PostgreSQL 上比較「應用程式時間」與「資料庫時間」。
// 情境是短網址服務 v2：每個短網址都有 expires_at，三個 API 節點同時判斷它過期了沒。
package dbclock

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DefaultDSN 對應 compose.yaml 的 port。
const DefaultDSN = "postgres://postgres:demo@127.0.0.1:55435/demo?sslmode=disable"

var (
	ErrNotFound = errors.New("link not found")
	ErrExpired  = errors.New("link expired")
)

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

// Reset 重建 links 資料表。時間欄位一律用 timestamptz，預設值交給資料庫的 now()。
func Reset(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`DROP TABLE IF EXISTS links`,
		`CREATE TABLE links (
			id         BIGSERIAL   PRIMARY KEY,
			code       TEXT        NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// Node 代表一台 API 節點。Skew 是這台機器的時鐘比真實時間快（正）或慢（負）多少，
// 用來模擬 NTP 偏移或虛擬機暫停後的漂移。
type Node struct {
	Name string
	Skew time.Duration
}

// Now 是這個節點「以為的現在」。
func (n Node) Now() time.Time { return time.Now().Add(n.Skew) }
