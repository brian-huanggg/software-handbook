// Package pgvsmysql 在 Postgres 與 MySQL 上重現隔離級別預設值、gap lock
// 與 MVCC 空間回收的差異。兩邊用同一張 links 資料表與同一組 SQL。
package pgvsmysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// 對應 compose.yaml 的 port。
const (
	DefaultPostgresDSN = "postgres://postgres:demo@127.0.0.1:55433/demo?sslmode=disable"
	DefaultMySQLDSN    = "root:demo@tcp(127.0.0.1:53307)/demo"
)

type Dialect int

const (
	Postgres Dialect = iota
	MySQL
)

func (d Dialect) String() string {
	if d == Postgres {
		return "PostgreSQL 18"
	}
	return "MySQL 8.4"
}

// Rebind 把 ? 佔位符換成該資料庫的寫法：Postgres 是 $1、$2，MySQL 維持 ?。
func (d Dialect) Rebind(q string) string {
	if d == MySQL {
		return q
	}
	var b strings.Builder
	n := 0
	for _, r := range q {
		if r == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Open 連上資料庫並確認連得到；連不上時回傳錯誤，呼叫端決定要略過還是失敗。
func Open(d Dialect, dsn string) (*sql.DB, error) {
	driver := "pgx"
	if d == MySQL {
		driver = "mysql"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", d, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", d, err)
	}
	return db, nil
}

// ResetLinks 重建 links 表。兩個資料庫用同一份 DDL。
func ResetLinks(ctx context.Context, db *sql.DB, d Dialect) error {
	stmts := []string{
		`DROP TABLE IF EXISTS links`,
		`CREATE TABLE links (
			id     BIGINT      NOT NULL PRIMARY KEY,
			code   VARCHAR(16) NOT NULL,
			clicks BIGINT      NOT NULL DEFAULT 0
		)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	if d == Postgres {
		// 關掉這張表的 autovacuum，才看得到 dead tuple 累積。
		if _, err := db.ExecContext(ctx, `ALTER TABLE links SET (autovacuum_enabled = false)`); err != nil {
			return err
		}
	}
	return nil
}
