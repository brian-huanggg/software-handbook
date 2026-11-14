-- 正規化到 3NF 之後。
CREATE TABLE plans (
    plan        TEXT PRIMARY KEY,
    daily_quota INT  NOT NULL CHECK (daily_quota > 0)
);

CREATE TABLE users (
    id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    plan  TEXT NOT NULL REFERENCES plans (plan)
);

CREATE TABLE links (
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code     VARCHAR(16) NOT NULL UNIQUE,
    url      TEXT        NOT NULL,
    owner_id BIGINT      NOT NULL REFERENCES users (id)
);

CREATE TABLE tags (
    name  TEXT PRIMARY KEY,
    color TEXT NOT NULL
);

CREATE TABLE link_tags (
    link_id BIGINT NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    tag     TEXT   NOT NULL REFERENCES tags (name),
    PRIMARY KEY (link_id, tag)
);
CREATE INDEX link_tags_tag_idx ON link_tags (tag, link_id);
