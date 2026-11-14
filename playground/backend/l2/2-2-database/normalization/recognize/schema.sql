-- AI 產生：短網址服務 v2 的資料表。
CREATE TABLE links (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id    BIGINT       NOT NULL REFERENCES users (id),
    owner_email TEXT         NOT NULL,
    code        VARCHAR(16)  NOT NULL UNIQUE,
    url         TEXT         NOT NULL,
    tags        TEXT         NOT NULL DEFAULT '',
    click_count BIGINT       NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX links_owner_idx ON links (owner_id);
