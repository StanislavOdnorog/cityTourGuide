-- Create ENUM types for POI
CREATE TYPE poi_type AS ENUM (
    'building',
    'street',
    'park',
    'monument',
    'church',
    'bridge',
    'square',
    'museum',
    'district',
    'other'
);

CREATE TYPE poi_status AS ENUM (
    'active',
    'disabled',
    'pending_review'
);

-- Create POI table
CREATE TABLE poi (
    id SERIAL PRIMARY KEY,
    city_id INTEGER NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    name_ru VARCHAR(255),
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    type poi_type NOT NULL DEFAULT 'other',
    tags JSONB DEFAULT '{}',
    address TEXT,
    interest_score SMALLINT NOT NULL DEFAULT 50,
    status poi_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create GIST index on location for spatial queries
CREATE INDEX idx_poi_location ON poi USING GIST (location);

-- Create composite index on (city_id, status) for filtered queries
CREATE INDEX idx_poi_city_status ON poi (city_id, status);
