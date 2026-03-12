-- Add heartbeat, worker tracking, and attempts columns for lease-based job recovery
ALTER TABLE inflation_job
    ADD COLUMN heartbeat_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN worker_id TEXT,
    ADD COLUMN attempts SMALLINT NOT NULL DEFAULT 0;

-- Add 'dead' status to the enum for jobs that exceed max retry attempts
ALTER TYPE inflation_job_status ADD VALUE IF NOT EXISTS 'dead';

-- Backfill running jobs with a stale heartbeat so they become reclaimable immediately
UPDATE inflation_job
SET heartbeat_at = started_at
WHERE status = 'running';
