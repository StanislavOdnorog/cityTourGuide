-- Create ENUM types for reports
CREATE TYPE report_type AS ENUM (
    'wrong_location',
    'wrong_fact',
    'inappropriate_content'
);

CREATE TYPE report_status AS ENUM (
    'new',
    'reviewed',
    'resolved',
    'dismissed'
);

-- Create reports table
CREATE TABLE report (
    id SERIAL PRIMARY KEY,
    story_id INTEGER NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type report_type NOT NULL,
    comment TEXT,
    user_lat DOUBLE PRECISION,
    user_lng DOUBLE PRECISION,
    status report_status NOT NULL DEFAULT 'new',
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Partial index for unresolved reports
CREATE INDEX idx_report_status_new ON report (status) WHERE status = 'new';

-- Index for story-level report lookups
CREATE INDEX idx_report_story ON report (story_id);
