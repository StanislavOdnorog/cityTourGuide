DROP INDEX IF EXISTS idx_cities_deleted_at;
ALTER TABLE cities DROP COLUMN IF EXISTS deleted_at;
