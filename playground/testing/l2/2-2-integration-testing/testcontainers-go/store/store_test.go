package store_test

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"handbook/playground/testing/testcontainers-go/store"
)

var (
	testDB  *sql.DB
	testRDB *redis.Client
)

// TestMain starts one PostgreSQL and one Redis for the whole package.
// Starting a container per test would cost seconds each time.
func TestMain(m *testing.M) {
	flag.Parse() // testing.Short() panics if called before the flags are parsed
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if testing.Short() {
		return m.Run() // tests below skip themselves; no Docker needed
	}
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:18",
		tcpostgres.WithDatabase("shortener"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("start postgres: %v", err)
		return 1
	}
	defer pg.Terminate(ctx) //nolint:errcheck // best effort; Ryuk reaps leftovers

	rd, err := tcredis.Run(ctx, "redis:8",
		testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
	)
	if err != nil {
		log.Printf("start redis: %v", err)
		return 1
	}
	defer rd.Terminate(ctx) //nolint:errcheck

	dsn, err := pg.ConnectionString(ctx)
	if err != nil {
		log.Printf("postgres dsn: %v", err)
		return 1
	}
	if testDB, err = sql.Open("pgx", dsn); err != nil {
		log.Printf("open db: %v", err)
		return 1
	}
	if _, err = testDB.ExecContext(ctx, store.Schema); err != nil {
		log.Printf("migrate: %v", err)
		return 1
	}

	uri, err := rd.ConnectionString(ctx)
	if err != nil {
		log.Printf("redis uri: %v", err)
		return 1
	}
	opts, err := redis.ParseURL(uri)
	if err != nil {
		log.Printf("parse redis url: %v", err)
		return 1
	}
	testRDB = redis.NewClient(opts)

	return m.Run()
}

// newStore gives each test a clean slate: truncate and flush before it runs.
func newStore(t *testing.T, ttl time.Duration) *store.Store {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}
	ctx := context.Background()
	if _, err := testDB.ExecContext(ctx, `TRUNCATE links`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := testRDB.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
	return store.New(testDB, testRDB, ttl)
}

func TestCreateDuplicateCode(t *testing.T) {
	s := newStore(t, time.Minute)
	ctx := context.Background()

	if err := s.Create(ctx, "abc123", "https://example.com/a"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := s.Create(ctx, "abc123", "https://example.com/b")
	if !errors.Is(err, store.ErrCodeTaken) {
		t.Fatalf("second create = %v, want ErrCodeTaken", err)
	}
}

// Many writers race for one code: the database must let exactly one win.
func TestCreateConcurrentSameCode(t *testing.T) {
	s := newStore(t, time.Minute)
	ctx := context.Background()

	const writers = 20
	var wins, taken int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Create(ctx, "race", "https://example.com/r")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, store.ErrCodeTaken):
				taken++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins != 1 || taken != writers-1 {
		t.Fatalf("wins=%d taken=%d, want 1 and %d", wins, taken, writers-1)
	}
}

func TestResolveFillsCache(t *testing.T) {
	s := newStore(t, time.Minute)
	ctx := context.Background()
	if err := s.Create(ctx, "hot", "https://example.com/hot"); err != nil {
		t.Fatal(err)
	}

	got, err := s.Resolve(ctx, "hot")
	if err != nil || got != "https://example.com/hot" {
		t.Fatalf("Resolve = %q, %v", got, err)
	}

	// Drop the row: a second Resolve can only succeed through Redis.
	if _, err := testDB.ExecContext(ctx, `DELETE FROM links WHERE code = 'hot'`); err != nil {
		t.Fatal(err)
	}
	got, err = s.Resolve(ctx, "hot")
	if err != nil || got != "https://example.com/hot" {
		t.Fatalf("cached Resolve = %q, %v", got, err)
	}
}

func TestResolveCacheExpires(t *testing.T) {
	s := newStore(t, 200*time.Millisecond)
	ctx := context.Background()
	if err := s.Create(ctx, "ttl", "https://example.com/ttl"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(ctx, "ttl"); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `DELETE FROM links WHERE code = 'ttl'`); err != nil {
		t.Fatal(err)
	}

	time.Sleep(400 * time.Millisecond) // real Redis TTL, no fake clock
	if _, err := s.Resolve(ctx, "ttl"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("after TTL, Resolve = %v, want ErrNotFound", err)
	}
}
