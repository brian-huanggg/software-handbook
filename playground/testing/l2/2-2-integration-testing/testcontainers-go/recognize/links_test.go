//go:build recognize

// AI 產生。不在預設的 go test ./... 範圍內；用 -tags recognize 才會編譯。
package recognize

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18",
			ExposedPorts: []string{"5432:5432"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "shortener",
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start container: %v", err)
	}

	time.Sleep(3 * time.Second)

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	db, err := sql.Open("pgx", "postgres://test:test@"+host+":5432/shortener?sslmode=disable")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertLink(t *testing.T) {
	db := setupDB(t)
	if _, err := db.Exec(`CREATE TABLE links (code text PRIMARY KEY, target text NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO links VALUES ('abc', 'https://example.com')`); err != nil {
		t.Fatal(err)
	}
}

func TestInsertDuplicateLink(t *testing.T) {
	db := setupDB(t)
	if _, err := db.Exec(`CREATE TABLE links (code text PRIMARY KEY, target text NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO links VALUES ('abc', 'https://example.com')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO links VALUES ('abc', 'https://example.org')`); err == nil {
		t.Fatal("want unique violation")
	}
}
