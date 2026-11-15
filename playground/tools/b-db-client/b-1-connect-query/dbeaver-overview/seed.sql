-- 短網址服務 v1 的簡化資料：links 與 clicks。
-- clicks.link_id 刻意不建索引，讓「小練習」的執行計畫有東西可看。
CREATE TABLE links (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       TEXT        NOT NULL UNIQUE,
    long_url   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE clicks (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    link_id    BIGINT      NOT NULL REFERENCES links (id),
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO links (code, long_url)
SELECT 'c' || g, 'https://example.com/page/' || g
FROM generate_series(1, 1000) AS g;

-- 50 萬筆點擊；link_id 1 刻意最熱門，約佔 10%。
INSERT INTO clicks (link_id, clicked_at)
SELECT CASE WHEN g % 10 = 0 THEN 1 ELSE 1 + (g % 1000) END,
       now() - (g || ' seconds')::interval
FROM generate_series(1, 500000) AS g;

ANALYZE;
