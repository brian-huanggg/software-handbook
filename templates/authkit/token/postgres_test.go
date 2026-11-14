package token_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"authkit/store"
	"authkit/token"
	"authkit/token/storetest"
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
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPostgresRefreshStore(t *testing.T) {
	pool := testPool(t)
	users := store.NewUserStore(pool)
	storetest.Run(t, func(t *testing.T) (token.RefreshStore, func() string) {
		newUser := func() string {
			u, err := users.Create(context.Background(), fmt.Sprintf("t-%s@example.com", uuid.NewString()), "h")
			if err != nil {
				t.Fatal(err)
			}
			return u.ID
		}
		return token.NewPostgresRefreshStore(pool), newUser
	})
}
