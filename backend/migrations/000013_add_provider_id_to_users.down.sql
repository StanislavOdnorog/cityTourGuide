DROP INDEX IF EXISTS idx_users_provider;

ALTER TABLE users DROP COLUMN IF EXISTS provider_id;
