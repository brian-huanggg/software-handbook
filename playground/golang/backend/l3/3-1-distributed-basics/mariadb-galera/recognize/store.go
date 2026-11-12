//go:build recognize

// Package recognize is AI-generated code for the review exercise.
package recognize

import (
	"context"
	"database/sql"
)

// Store persists short links. db points at the load balancer in front of
// all three Galera nodes, which spreads connections evenly.
type Store struct{ db *sql.DB }

const schema = `
CREATE TABLE links (
  code VARCHAR(8) PRIMARY KEY,
  url  TEXT NOT NULL,
  hits BIGINT NOT NULL DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE click_log (
  code VARCHAR(8),
  at   DATETIME
) ENGINE=MyISAM;`

func (s *Store) Create(ctx context.Context, code, url string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO links (code, url) VALUES (?, ?)", code, url)
	return err
}

// Resolve is called right after Create to render the "your link" page.
func (s *Store) Resolve(ctx context.Context, code string) (string, error) {
	var url string
	err := s.db.QueryRowContext(ctx,
		"SELECT url FROM links WHERE code = ?", code).Scan(&url)
	return url, err
}

func (s *Store) Hit(ctx context.Context, code string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"UPDATE links SET hits = hits + 1 WHERE code = ?", code); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO click_log (code, at) VALUES (?, NOW())", code); err != nil {
		return err
	}
	tx.Commit()
	return nil
}
