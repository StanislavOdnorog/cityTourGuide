DROP INDEX IF EXISTS idx_users_deletion_scheduled;

ALTER TABLE users
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deletion_scheduled_at;
