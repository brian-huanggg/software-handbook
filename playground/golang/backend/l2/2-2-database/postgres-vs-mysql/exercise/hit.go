package exercise

import (
	"context"
	"database/sql"
	"errors"

	pgvsmysql "handbook/playground/golang/backend/postgres-vs-mysql"
)

// Hit 記一次點擊：code 不存在就建立並回傳 1，存在就加 1，回傳加完之後的值。
// 表是 hits(code VARCHAR(16) PRIMARY KEY, clicks BIGINT NOT NULL)。
//
// 要求：同時有很多個 goroutine 呼叫 Hit，每個呼叫拿到的值都不同，
// 最後的總數等於呼叫次數；同一份程式要在 Postgres 與 MySQL 上都通過。
func Hit(ctx context.Context, db *sql.DB, d pgvsmysql.Dialect, code string) (int64, error) {
	return 0, errors.New("not implemented")
}
