-- Absolute lifetime of a login chain. Rotation never extends a token past it.
ALTER TABLE refresh_tokens ADD COLUMN family_expires_at timestamptz;
UPDATE refresh_tokens SET family_expires_at = expires_at;
ALTER TABLE refresh_tokens ALTER COLUMN family_expires_at SET NOT NULL;
