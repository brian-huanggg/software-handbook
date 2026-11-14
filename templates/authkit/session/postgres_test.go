package session_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brian-huanggg/software-handbook/templates/authkit/session"
	"github.com/brian-huanggg/software-handbook/templates/authkit/session/storetest"
	"github.com/brian-huanggg/software-handbook/templates/authkit/store"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable"
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

func TestPostgresStore(t *testing.T) {
	pool := testPool(t)
	users := store.NewUserStore(pool)
	storetest.Run(t, func(t *testing.T) (session.Store, func() string) {
		newUser := func() string {
			u, err := users.Create(context.Background(), fmt.Sprintf("s-%s@example.com", uuid.NewString()), "h")
			if err != nil {
				t.Fatal(err)
			}
			return u.ID
		}
		return session.NewPostgresStore(pool), newUser
	})
}
