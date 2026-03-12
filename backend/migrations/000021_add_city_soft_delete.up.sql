ALTER TABLE cities ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Public queries filter on deleted_at IS NULL; index supports that efficiently.
CREATE INDEX idx_cities_deleted_at ON cities (deleted_at) WHERE deleted_at IS NOT NULL;
