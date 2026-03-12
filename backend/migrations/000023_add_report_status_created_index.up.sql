-- Covers report queries that filter by status and order by created_at DESC
-- (e.g. ListAdmin with created_at sort, GetByPOIID).
CREATE INDEX IF NOT EXISTS idx_report_status_created
    ON report (status, created_at DESC);
