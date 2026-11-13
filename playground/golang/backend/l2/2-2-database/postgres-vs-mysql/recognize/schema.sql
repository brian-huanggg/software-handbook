-- AI 產生：短網址服務 v2 的 PostgreSQL 資料表
CREATE TABLE links (
    id         BIGSERIAL    PRIMARY KEY,
    owner_id   BIGINT       NOT NULL,
    code       VARCHAR(16)  NOT NULL,
    url        TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_links_code ON links (code);
