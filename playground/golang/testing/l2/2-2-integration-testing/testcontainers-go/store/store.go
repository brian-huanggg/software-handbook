// Package store is a tiny slice of the short-URL service v2: links live in
// PostgreSQL, hot lookups are cached in Redis.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

var (
	ErrCodeTaken = errors.New("store: code already taken")
	ErrNotFound  = errors.New("store: link not found")
)

const Schema = `
CREATE TABLE IF NOT EXISTS links (
	code       text PRIMARY KEY,
	target     text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
)`

type Store struct {
	db       *sql.DB
	rdb      *redis.Client
	cacheTTL time.Duration
}

func New(db *sql.DB, rdb *redis.Client, cacheTTL time.Duration) *Store {
	return &Store{db: db, rdb: rdb, cacheTTL: cacheTTL}
}

// Create inserts a link. The primary key, not an application-level
// check-then-insert, is what decides who wins a race on the same code.
func (s *Store) Create(ctx context.Context, code, target string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO links (code, target) VALUES ($1, $2)`, code, target)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrCodeTaken
	}
	if err != nil {
		return fmt.Errorf("insert link: %w", err)
	}
	return nil
}

// Resolve looks in Redis first and falls back to PostgreSQL, then fills the cache.
func (s *Store) Resolve(ctx context.Context, code string) (string, error) {
	target, err := s.rdb.Get(ctx, cacheKey(code)).Result()
	if err == nil {
		return target, nil
	}
	if !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("cache get: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT target FROM links WHERE code = $1`, code).Scan(&target)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("select link: %w", err)
	}

	if err := s.rdb.Set(ctx, cacheKey(code), target, s.cacheTTL).Err(); err != nil {
		return "", fmt.Errorf("cache set: %w", err)
	}
	return target, nil
}

func cacheKey(code string) string { return "link:" + code }
