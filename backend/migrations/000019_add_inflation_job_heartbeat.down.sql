ALTER TABLE inflation_job
    DROP COLUMN IF EXISTS heartbeat_at,
    DROP COLUMN IF EXISTS worker_id,
    DROP COLUMN IF EXISTS attempts;

-- Note: PostgreSQL does not support removing values from an enum type.
-- The 'dead' value will remain in the enum but is harmless.
