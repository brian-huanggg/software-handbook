package pgvsmysql

import (
	"context"
	"database/sql"
	"fmt"
)

// Stage 是某個時間點，links 表（含索引）占用的位元組數。
type Stage struct {
	Label string
	Bytes int64
}

// SpaceAfterUpdates 載入 rows 筆資料，把每一列的 clicks 全部加 1，重複 rounds 次，
// 並在每個階段量測資料表加索引的大小。Postgres 會多一個 VACUUM 階段。
func SpaceAfterUpdates(ctx context.Context, db *sql.DB, d Dialect, rows, rounds int) ([]Stage, error) {
	if err := ResetLinks(ctx, db, d); err != nil {
		return nil, err
	}
	if err := load(ctx, db, d, rows); err != nil {
		return nil, err
	}

	var stages []Stage
	measure := func(label string) error {
		n, err := tableBytes(ctx, db, d)
		if err != nil {
			return err
		}
		stages = append(stages, Stage{label, n})
		return nil
	}
	if err := measure("載入後"); err != nil {
		return nil, err
	}

	for i := 1; i <= rounds; i++ {
		if _, err := db.ExecContext(ctx, `UPDATE links SET clicks = clicks + 1`); err != nil {
			return nil, err
		}
		if err := measure(fmt.Sprintf("全表 UPDATE 第 %d 次後", i)); err != nil {
			return nil, err
		}
	}

	if d == Postgres {
		if _, err := db.ExecContext(ctx, `VACUUM links`); err != nil {
			return nil, err
		}
		if err := measure("VACUUM 後"); err != nil {
			return nil, err
		}
		if _, err := db.ExecContext(ctx, `UPDATE links SET clicks = clicks + 1`); err != nil {
			return nil, err
		}
		if err := measure("VACUUM 後再 UPDATE 一次"); err != nil {
			return nil, err
		}
	}
	return stages, nil
}

func load(ctx context.Context, db *sql.DB, d Dialect, rows int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, d.Rebind(`INSERT INTO links (id, code) VALUES (?, ?)`))
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i := 1; i <= rows; i++ {
		if _, err := stmt.ExecContext(ctx, i, fmt.Sprintf("c%07d", i)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func tableBytes(ctx context.Context, db *sql.DB, d Dialect) (int64, error) {
	var n int64
	if d == Postgres {
		err := db.QueryRowContext(ctx, `SELECT pg_total_relation_size('links')`).Scan(&n)
		return n, err
	}
	// MySQL 的統計值有快取，先重新分析並關掉快取。
	if _, err := db.ExecContext(ctx, `ANALYZE TABLE links`); err != nil {
		return 0, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SET SESSION information_schema_stats_expiry = 0`); err != nil {
		return 0, err
	}
	err = conn.QueryRowContext(ctx,
		`SELECT data_length + index_length FROM information_schema.tables
		 WHERE table_schema = DATABASE() AND table_name = 'links'`).Scan(&n)
	return n, err
}
