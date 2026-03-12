-- Report cursor-pagination index: covers List and ListAdmin queries
-- that filter by status and paginate with id > cursor ORDER BY id ASC.
-- The existing idx_report_status_new partial index only covers status = 'new';
-- this composite covers all status values with efficient cursor seeks.
CREATE INDEX IF NOT EXISTS idx_report_status_id
    ON report (status, id);

-- Story POI-lookup index: covers GetByPOIID, ListByPOIID, and
-- GetDownloadManifest queries that filter by (poi_id, language, status)
-- and sort by (order_index, created_at). Extends the existing
-- idx_story_poi_language_status to include sort columns, eliminating
-- post-filter sorts on large result sets.
CREATE INDEX IF NOT EXISTS idx_story_poi_lang_status_order
    ON story (poi_id, language, status, order_index, created_at);
