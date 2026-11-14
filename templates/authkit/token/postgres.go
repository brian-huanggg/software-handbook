package token

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRefreshStore struct{ pool *pgxpool.Pool }

// NewPostgresRefreshStore returns a RefreshStore backed by the refresh_tokens table.
func NewPostgresRefreshStore(pool *pgxpool.Pool) RefreshStore { return &pgRefreshStore{pool: pool} }

func (s *pgRefreshStore) Create(ctx context.Context, r Refresh) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id_hash, user_id, family_id, used_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		r.IDHash, r.UserID, r.FamilyID, r.UsedAt, r.ExpiresAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func (s *pgRefreshStore) Get(ctx context.Context, idHash []byte) (Refresh, error) {
	var r Refresh
	err := s.pool.QueryRow(ctx,
		`SELECT id_hash, user_id::text, family_id::text, used_at, expires_at
		 FROM refresh_tokens WHERE id_hash = $1`, idHash).
		Scan(&r.IDHash, &r.UserID, &r.FamilyID, &r.UsedAt, &r.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Refresh{}, ErrNotFound
	}
	return r, err
}

// MarkUsed is a single conditional UPDATE, so concurrent callers are
// serialised by the row lock and only one sees RowsAffected() == 1.
func (s *pgRefreshStore) MarkUsed(ctx context.Context, idHash []byte, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET used_at = $2 WHERE id_hash = $1 AND used_at IS NULL`,
		idHash, at)
	return tag.RowsAffected() == 1, err
}

func (s *pgRefreshStore) DeleteFamily(ctx context.Context, familyID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE family_id = $1`, familyID)
	return err
}
