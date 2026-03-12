CREATE TABLE orphan_cleanup_job (
    id          SERIAL PRIMARY KEY,
    object_key  TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed')),
    attempts    INT NOT NULL DEFAULT 0,
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orphan_cleanup_job_status ON orphan_cleanup_job (status) WHERE status = 'pending';
