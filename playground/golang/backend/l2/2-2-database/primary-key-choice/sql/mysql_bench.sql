-- InnoDB 版對照：主鍵就是 clustered index，secondary index 的每個 entry 都帶一份主鍵。
-- N = 1000 * 1000 * 2，用三個 seq 相乘產生資料列。
SET SESSION cte_max_recursion_depth = 2000;
DROP TABLE IF EXISTS seq, links_bigint, links_rand16, links_uuid1_swap, result;

CREATE TABLE seq (n INT PRIMARY KEY);
INSERT INTO seq WITH RECURSIVE r (n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM r WHERE n < 1000) SELECT n FROM r;

CREATE TABLE links_bigint     (id BIGINT AUTO_INCREMENT PRIMARY KEY, code CHAR(32) NOT NULL, url VARCHAR(100) NOT NULL, KEY (code));
CREATE TABLE links_rand16     (id BINARY(16) PRIMARY KEY,            code CHAR(32) NOT NULL, url VARCHAR(100) NOT NULL, KEY (code));
CREATE TABLE links_uuid1_swap (id BINARY(16) PRIMARY KEY,            code CHAR(32) NOT NULL, url VARCHAR(100) NOT NULL, KEY (code));
CREATE TABLE result (variant VARCHAR(20), seconds DECIMAL(8,2));

SET @n0 = 0;

-- 1) BIGINT AUTO_INCREMENT
SET @t0 = NOW(6);
INSERT INTO links_bigint (code, url)
SELECT MD5(a.n * 1000000 + b.n * 1000 + c.n), CONCAT('https://example.com/', MD5(a.n + b.n * c.n), MD5(a.n * c.n + b.n))
FROM seq a, seq b, seq c WHERE c.n <= 2;
INSERT INTO result VALUES ('links_bigint', TIMESTAMPDIFF(MICROSECOND, @t0, NOW(6)) / 1e6);

-- 2) 隨機 16 bytes：等同 UUID v4 的排序行為（RAND() 只有約 2^30 種狀態，兩百萬筆會撞，所以改用 MD5 雜湊）
SET @t0 = NOW(6);
INSERT INTO links_rand16 (id, code, url)
SELECT UNHEX(MD5(CONCAT('id-', a.n, '-', b.n, '-', c.n))), MD5(a.n * 1000000 + b.n * 1000 + c.n), CONCAT('https://example.com/', MD5(a.n + b.n * c.n), MD5(a.n * c.n + b.n))
FROM seq a, seq b, seq c WHERE c.n <= 2;
INSERT INTO result VALUES ('links_rand16', TIMESTAMPDIFF(MICROSECOND, @t0, NOW(6)) / 1e6);

-- 3) UUID v1 + UUID_TO_BIN(uuid, 1)：把時間欄位換到前面，變成遞增
SET @t0 = NOW(6);
INSERT INTO links_uuid1_swap (id, code, url)
SELECT UUID_TO_BIN(UUID(), 1), MD5(a.n * 1000000 + b.n * 1000 + c.n), CONCAT('https://example.com/', MD5(a.n + b.n * c.n), MD5(a.n * c.n + b.n))
FROM seq a, seq b, seq c WHERE c.n <= 2;
INSERT INTO result VALUES ('links_uuid1_swap', TIMESTAMPDIFF(MICROSECOND, @t0, NOW(6)) / 1e6);

SELECT * FROM result;

ANALYZE TABLE links_bigint, links_rand16, links_uuid1_swap;
SELECT table_name,
       ROUND(data_length / 1024 / 1024)  AS clustered_mb,
       ROUND(index_length / 1024 / 1024) AS secondary_idx_mb
FROM information_schema.tables
WHERE table_schema = 'demo' AND table_name LIKE 'links_%'
ORDER BY table_name;
