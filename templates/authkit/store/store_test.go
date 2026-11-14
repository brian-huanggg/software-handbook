package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brian-huanggg/software-handbook/templates/authkit/store"
)

const defaultURL = "postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable"

// testPool connects to the compose DB (run `make up migrate-up`) and skips
// the test when it is unreachable. Tests use unique emails instead of
// truncating, so packages can run in parallel.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = defaultURL
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err == nil {
		err = pool.Ping(ctx)
	}
	if err != nil {
		// A green CI run must mean Postgres was actually tested.
		if os.Getenv("CI") != "" {
			t.Fatalf("database unavailable in CI: %v", err)
		}
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func uniqueEmail() string { return fmt.Sprintf("u-%s@example.com", uuid.NewString()) }

func TestCreate_ThenLookup(t *testing.T) {
	s := store.NewUserStore(testPool(t))
	ctx := context.Background()
	email := uniqueEmail()

	u, err := s.Create(ctx, email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || u.Email != email || u.PasswordHash != "hash" || u.Role != "user" || u.CreatedAt.IsZero() {
		t.Fatalf("unexpected user: %+v", u)
	}

	byEmail, err := s.ByEmail(ctx, email)
	if err != nil || byEmail.ID != u.ID {
		t.Fatalf("ByEmail = %+v, %v", byEmail, err)
	}
	byID, err := s.ByID(ctx, u.ID)
	if err != nil || byID.Email != email {
		t.Fatalf("ByID = %+v, %v", byID, err)
	}
}

func TestCreate_DuplicateEmail(t *testing.T) {
	s := store.NewUserStore(testPool(t))
	ctx := context.Background()
	email := uniqueEmail()

	if _, err := s.Create(ctx, email, "h"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, email, "h"); !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("exact duplicate: got %v, want ErrEmailTaken", err)
	}
	if _, err := s.Create(ctx, strings.ToUpper(email), "h"); !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("case-different duplicate: got %v, want ErrEmailTaken", err)
	}
}

func TestByEmail_CaseInsensitive(t *testing.T) {
	s := store.NewUserStore(testPool(t))
	ctx := context.Background()
	email := uniqueEmail()
	u, _ := s.Create(ctx, email, "h")

	got, err := s.ByEmail(ctx, strings.ToUpper(email))
	if err != nil || got.ID != u.ID {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestNotFound(t *testing.T) {
	s := store.NewUserStore(testPool(t))
	ctx := context.Background()

	if _, err := s.ByEmail(ctx, uniqueEmail()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("ByEmail: got %v, want ErrNotFound", err)
	}
	if _, err := s.ByID(ctx, uuid.NewString()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("ByID unknown: got %v, want ErrNotFound", err)
	}
	if _, err := s.ByID(ctx, "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("ByID malformed: got %v, want ErrNotFound", err)
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	s := store.NewUserStore(testPool(t))
	_, err := s.Create(context.Background(), "not-an-email", "h")
	if !errors.Is(err, store.ErrInvalidEmail) {
		t.Fatalf("got %v, want ErrInvalidEmail", err)
	}
}

func TestReplacePasswordHash(t *testing.T) {
	s := store.NewUserStore(testPool(t))
	ctx := context.Background()
	u, err := s.Create(ctx, uniqueEmail(), "old")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.ReplacePasswordHash(ctx, u.ID, "stale", "new"); err != nil || ok {
		t.Fatalf("stale old hash: ok=%v err=%v, want false, nil", ok, err)
	}
	if ok, err := s.ReplacePasswordHash(ctx, u.ID, "old", "new"); err != nil || !ok {
		t.Fatalf("matching old hash: ok=%v err=%v", ok, err)
	}
	if got, _ := s.ByID(ctx, u.ID); got.PasswordHash != "new" {
		t.Fatalf("hash = %q, want new", got.PasswordHash)
	}
}
