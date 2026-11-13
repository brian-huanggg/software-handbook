// 在 Postgres 與 MySQL 上各跑一次三個實驗並印出結果。先執行 make up。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pgvsmysql "handbook/playground/backend/postgres-vs-mysql"
)

func main() {
	ctx := context.Background()

	targets := []struct {
		d   pgvsmysql.Dialect
		dsn string
	}{
		{pgvsmysql.Postgres, envOr("PG_DSN", pgvsmysql.DefaultPostgresDSN)},
		{pgvsmysql.MySQL, envOr("MYSQL_DSN", pgvsmysql.DefaultMySQLDSN)},
	}

	for _, t := range targets {
		db, err := pgvsmysql.Open(t.d, t.dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		fmt.Printf("== %s ==\n", t.d)

		level, err := pgvsmysql.DefaultIsolation(ctx, db, t.d)
		must(err)
		fmt.Printf("預設隔離級別：%s\n", level)

		first, second, err := pgvsmysql.NonRepeatableRead(ctx, db, t.d)
		must(err)
		fmt.Printf("同一個交易內數兩次筆數：%d → %d\n", first, second)

		blocked, err := pgvsmysql.GapInsert(ctx, db, t.d, 2*time.Second)
		must(err)
		fmt.Printf("別人鎖住 id 15–25 的空範圍後，插入 id 20 是否被擋住：%v\n", blocked)

		stages, err := pgvsmysql.SpaceAfterUpdates(ctx, db, t.d, 200_000, 2)
		must(err)
		for _, s := range stages {
			fmt.Printf("  %-22s %6.1f MB\n", s.Label, float64(s.Bytes)/(1<<20))
		}
		fmt.Println()
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
