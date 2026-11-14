-- 000001_init_auth_up.sql
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text UNIQUE NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL DEFAULT 'user',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz
);
CREATE TABLE sessions (
    id_hash bytea PRIMARY KEY, -- SHA-256 (session id)
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);
CREATE TABLE refresh_tokens (
    id_hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    family_id uuid NOT NULL, -- same login chain
    used_at timestamptz,
    expires_at timestamptz NOT NULL
);
