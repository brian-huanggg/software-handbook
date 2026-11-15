//go:build ignore

// AI 產生：短網址服務 v2 的「建立短網址」與「清掉過期短網址」。資料庫是 PostgreSQL，有三台 API 節點。
//
//	CREATE TABLE links (
//	    id         BIGSERIAL PRIMARY KEY,
//	    code       TEXT NOT NULL UNIQUE,
//	    expires_at TIMESTAMPTZ NOT NULL,
//	    status     TEXT NOT NULL DEFAULT 'active'
//	)
package main

import (
	"context"
	"database/sql"
	"time"
)

type Server struct{ db *sql.DB }

func (s *Server) Create(ctx context.Context, code string, ttl time.Duration) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO links (code, expires_at) VALUES ($1, $2)`,
		code, time.Now().Add(ttl))
	return err
}

func (s *Server) Resolve(ctx context.Context, code string) (string, error) {
	var status string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM links WHERE code = $1`, code).Scan(&status)
	if err != nil {
		return "", err
	}
	if status != "active" {
		return "", sql.ErrNoRows
	}
	return code, nil
}

// Sweep 每分鐘由每台節點執行一次。
func (s *Server) Sweep(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE links SET status = 'expired' WHERE expires_at < $1 AND status = 'active'`,
		time.Now())
	return err
}

// Audit 掃描所有短網址，並回報這一批掃了多久。
func (s *Server) Audit(ctx context.Context) (time.Duration, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var started time.Time
	if err := tx.QueryRowContext(ctx, `SELECT now()`).Scan(&started); err != nil {
		return 0, err
	}
	// ... 逐批掃描 ...
	var finished time.Time
	if err := tx.QueryRowContext(ctx, `SELECT now()`).Scan(&finished); err != nil {
		return 0, err
	}
	return finished.Sub(started), tx.Commit()
}
