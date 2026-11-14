// Package store persists users in Postgres via pgx.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmailTaken   = errors.New("store: email already registered")
	ErrInvalidEmail = errors.New("store: invalid email")
	ErrNotFound     = errors.New("store: not found")
)

// Postgres error codes.
const (
	pgUniqueViolation  = "23505"
	pgCheckViolation   = "23514"
	pgInvalidTextRepr  = "22P02" // e.g. malformed uuid
	emailFormatCheckNm = "users_email_format"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type UserStore interface {
	// Create inserts a user with the default role. Returns ErrEmailTaken
	// (case-insensitive) or ErrInvalidEmail.
	Create(ctx context.Context, email, passwordHash string) (User, error)
	ByEmail(ctx context.Context, email string) (User, error)
	ByID(ctx context.Context, id string) (User, error)
}

type pgUserStore struct{ pool *pgxpool.Pool }

func NewUserStore(pool *pgxpool.Pool) UserStore { return &pgUserStore{pool: pool} }

const userCols = `id::text, email::text, password_hash, role, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	return u, err
}

func (s *pgUserStore) Create(ctx context.Context, email, passwordHash string) (User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING `+userCols,
		email, passwordHash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch {
			case pgErr.Code == pgUniqueViolation:
				return User{}, ErrEmailTaken
			case pgErr.Code == pgCheckViolation && pgErr.ConstraintName == emailFormatCheckNm:
				return User{}, ErrInvalidEmail
			}
		}
		return User{}, err
	}
	return u, nil
}

func (s *pgUserStore) ByEmail(ctx context.Context, email string) (User, error) {
	return s.one(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email)
}

func (s *pgUserStore) ByID(ctx context.Context, id string) (User, error) {
	return s.one(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id)
}

func (s *pgUserStore) one(ctx context.Context, q string, arg any) (User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, q, arg))
	if err == nil {
		return u, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgInvalidTextRepr {
		return User{}, ErrNotFound
	}
	return User{}, err
}
