-- Partial GIST index on active POIs only; replaces the unfiltered idx_poi_location
-- so spatial scans skip inactive rows.
CREATE INDEX idx_poi_location_active ON poi USING GIST (location) WHERE status = 'active';
DROP INDEX idx_poi_location;
