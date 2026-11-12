-- 三種主鍵各插入 N 筆，比較耗時、WAL 量與索引大小。
-- 欄位模擬短網址服務 v2 的 links 表：主鍵、短碼、原始網址、建立時間。
-- N = 2,000,000（psql 變數不會展開進 DO 區塊，所以直接寫在下面）
\timing off
CREATE EXTENSION IF NOT EXISTS pgstattuple;

DROP TABLE IF EXISTS links_bigint, links_uuid4, links_uuid7;

CREATE TABLE links_bigint (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, code text NOT NULL, url text NOT NULL);
CREATE TABLE links_uuid4  (id uuid DEFAULT gen_random_uuid() PRIMARY KEY,      code text NOT NULL, url text NOT NULL);
CREATE TABLE links_uuid7  (id uuid DEFAULT uuidv7() PRIMARY KEY,               code text NOT NULL, url text NOT NULL);
CREATE INDEX ON links_bigint (code);
CREATE INDEX ON links_uuid4  (code);
CREATE INDEX ON links_uuid7  (code);

CREATE TEMP TABLE result (variant text, seconds numeric, wal_mb numeric);

DO $$
DECLARE
  t text; lsn0 pg_lsn; t0 timestamptz;
BEGIN
  FOREACH t IN ARRAY ARRAY['links_bigint', 'links_uuid4', 'links_uuid7'] LOOP
    CHECKPOINT;  -- 每種從乾淨狀態開始，避免 full-page write 的次數互相影響
    lsn0 := pg_current_wal_lsn();
    t0 := clock_timestamp();
    EXECUTE format(
      'INSERT INTO %I (code, url) SELECT md5(g::text), ''https://example.com/'' || md5((g*7)::text) || md5((g*13)::text) FROM generate_series(1, 2000000) g',
      t);
    INSERT INTO result VALUES (
      t,
      round(extract(epoch FROM clock_timestamp() - t0)::numeric, 2),
      round((pg_current_wal_lsn() - lsn0) / 1024.0 / 1024.0, 0));
  END LOOP;
END $$;

SELECT variant AS "variant", seconds AS "insert_sec", wal_mb AS "wal_mb" FROM result ORDER BY variant;

SELECT 'links_bigint' AS "table", pg_size_pretty(pg_relation_size('links_bigint_pkey')) AS "pk_index",
       (pgstatindex('links_bigint_pkey')).avg_leaf_density AS "leaf_density_%"
UNION ALL
SELECT 'links_uuid4', pg_size_pretty(pg_relation_size('links_uuid4_pkey')), (pgstatindex('links_uuid4_pkey')).avg_leaf_density
UNION ALL
SELECT 'links_uuid7', pg_size_pretty(pg_relation_size('links_uuid7_pkey')), (pgstatindex('links_uuid7_pkey')).avg_leaf_density;

-- Postgres 的 secondary index 存的是 ctid（6 bytes），不是主鍵，所以 code 索引大小應該差不多。
SELECT 'links_bigint' AS "table", pg_size_pretty(pg_relation_size('links_bigint_code_idx')) AS "code_index"
UNION ALL SELECT 'links_uuid4', pg_size_pretty(pg_relation_size('links_uuid4_code_idx'))
UNION ALL SELECT 'links_uuid7', pg_size_pretty(pg_relation_size('links_uuid7_code_idx'));
