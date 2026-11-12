-- AI 產生：短網址服務 v2 的 MySQL 資料表
CREATE TABLE links (
    id         CHAR(36)     NOT NULL PRIMARY KEY,
    owner_id   BIGINT       NOT NULL,
    code       VARCHAR(16)  NOT NULL,
    url        TEXT         NOT NULL,
    created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_code (code),
    KEY idx_owner (owner_id, created_at)
) ENGINE=InnoDB;
