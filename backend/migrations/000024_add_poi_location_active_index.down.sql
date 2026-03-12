CREATE INDEX idx_poi_location ON poi USING GIST (location);
DROP INDEX idx_poi_location_active;
