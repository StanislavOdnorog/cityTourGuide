-- Create ENUM types for inflation jobs
CREATE TYPE inflation_job_status AS ENUM (
    'pending',
    'running',
    'completed',
    'failed'
);

CREATE TYPE inflation_trigger_type AS ENUM (
    'user_proximity',
    'admin_manual'
);

-- Create inflation_jobs table
CREATE TABLE inflation_job (
    id SERIAL PRIMARY KEY,
    poi_id INTEGER NOT NULL REFERENCES poi(id) ON DELETE CASCADE,
    status inflation_job_status NOT NULL DEFAULT 'pending',
    trigger_type inflation_trigger_type NOT NULL,
    segments_count SMALLINT NOT NULL DEFAULT 0,
    max_segments SMALLINT NOT NULL DEFAULT 3,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_log TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for pending jobs processing
CREATE INDEX idx_inflation_job_status ON inflation_job (status) WHERE status IN ('pending', 'running');

-- Index for POI-level lookups
CREATE INDEX idx_inflation_job_poi ON inflation_job (poi_id);
