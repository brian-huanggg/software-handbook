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
	_, err := s.pool.Exec(ctx, insertRefresh,
		r.IDHash, r.UserID, r.FamilyID, r.UsedAt, r.ExpiresAt, r.FamilyExpiresAt)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (s *pgRefreshStore) Get(ctx context.Context, idHash []byte) (Refresh, error) {
	var r Refresh
	err := s.pool.QueryRow(ctx,
		`SELECT id_hash, user_id::text, family_id::text, used_at, expires_at, family_expires_at
		 FROM refresh_tokens WHERE id_hash = $1`, idHash).
		Scan(&r.IDHash, &r.UserID, &r.FamilyID, &r.UsedAt, &r.ExpiresAt, &r.FamilyExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Refresh{}, ErrNotFound
	}
	return r, err
}

const insertRefresh = `INSERT INTO refresh_tokens (id_hash, user_id, family_id, used_at, expires_at, family_expires_at)
	 VALUES ($1, $2, $3, $4, $5, $6)`

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Rotate runs the conditional UPDATE and the INSERT in one transaction.
// Concurrent callers are serialised by the row lock, so only one sees
// RowsAffected() == 1; if the INSERT fails, the UPDATE is rolled back.
func (s *pgRefreshStore) Rotate(ctx context.Context, oldHash []byte, usedAt time.Time, next Refresh) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit

	tag, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET used_at = $2 WHERE id_hash = $1 AND used_at IS NULL`,
		oldHash, usedAt)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() != 1 {
		return false, nil
	}
	_, err = tx.Exec(ctx, insertRefresh,
		next.IDHash, next.UserID, next.FamilyID, next.UsedAt, next.ExpiresAt, next.FamilyExpiresAt)
	if isUniqueViolation(err) {
		return false, ErrDuplicate
	}
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (s *pgRefreshStore) DeleteFamily(ctx context.Context, familyID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE family_id = $1`, familyID)
	return err
}

func (s *pgRefreshStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	// expires_at never passes family_expires_at, so one indexed check covers both.
	tag, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < $1`, now)
	return tag.RowsAffected(), err
}
