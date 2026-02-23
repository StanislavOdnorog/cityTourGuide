-- Add soft delete support for GDPR compliance (TASK-056)
ALTER TABLE users
    ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN deletion_scheduled_at TIMESTAMP WITH TIME ZONE;

-- Index for efficient lookup of accounts pending hard deletion
CREATE INDEX idx_users_deletion_scheduled
    ON users (deletion_scheduled_at)
    WHERE deletion_scheduled_at IS NOT NULL AND deleted_at IS NULL;
