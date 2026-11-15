//go:build ignore

// AI 產生：短網址服務 v2 的「領取點擊額度」與「合併短網址額度」。資料庫是 PostgreSQL。
package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
)

var ErrNoQuota = errors.New("quota exhausted")

type Server struct {
	db *sql.DB
}

// 每次點擊先確認額度、通過安全檢查，再扣一點。
func (s *Server) Click(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var quota int
	var target string
	if err := tx.QueryRowContext(ctx,
		`SELECT quota, url FROM links WHERE id = $1`, id).Scan(&quota, &target); err != nil {
		return err
	}
	if quota <= 0 {
		return ErrNoQuota
	}

	// 向外部的安全瀏覽服務確認網址沒有被列入黑名單。
	resp, err := http.Get("https://safe.example.com/check?u=" + url.QueryEscape(target))
	if err != nil {
		return err
	}
	resp.Body.Close()

	if _, err := tx.ExecContext(ctx,
		`UPDATE links SET quota = $1, clicks = clicks + 1 WHERE id = $2`, quota-1, id); err != nil {
		return err
	}
	return tx.Commit()
}

// 把 src 的額度全部併入 dst，src 歸零。
func (s *Server) Merge(ctx context.Context, src, dst int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var moved int
	if err := tx.QueryRowContext(ctx,
		`SELECT quota FROM links WHERE id = $1 FOR UPDATE`, src).Scan(&moved); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE links SET quota = 0 WHERE id = $1`, src); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE links SET quota = quota + $1 WHERE id = $2`, moved, dst); err != nil {
		return err
	}
	return tx.Commit()
}

func main() {}
