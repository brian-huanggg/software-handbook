package session

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgStore struct{ pool *pgxpool.Pool }

// NewPostgresStore returns a Store backed by the sessions table.
func NewPostgresStore(pool *pgxpool.Pool) Store { return &pgStore{pool: pool} }

func (s *pgStore) Create(ctx context.Context, sess Session) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (id_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		sess.IDHash, sess.UserID, sess.ExpiresAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func (s *pgStore) Get(ctx context.Context, idHash []byte) (Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx,
		`SELECT id_hash, user_id::text, expires_at FROM sessions WHERE id_hash = $1`, idHash).
		Scan(&sess.IDHash, &sess.UserID, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return sess, err
}

func (s *pgStore) Delete(ctx context.Context, idHash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id_hash = $1`, idHash)
	return err
}

func (s *pgStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < $1`, now)
	return tag.RowsAffected(), err
}
