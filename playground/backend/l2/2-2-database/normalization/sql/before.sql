-- 短網址服務 v2 的第一版資料表：什麼都放在 links。
CREATE TABLE links (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code        VARCHAR(16)  NOT NULL UNIQUE,
    url         TEXT         NOT NULL,
    tags        TEXT         NOT NULL DEFAULT '',   -- 'go,db,tutorial'
    owner_id    BIGINT       NOT NULL,
    owner_email TEXT         NOT NULL,
    plan        TEXT         NOT NULL,
    daily_quota INT          NOT NULL
);
