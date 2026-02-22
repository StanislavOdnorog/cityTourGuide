-- Create ENUM types for Story
CREATE TYPE story_layer_type AS ENUM (
    'atmosphere',
    'human_story',
    'hidden_detail',
    'time_shift',
    'general'
);

CREATE TYPE story_status AS ENUM (
    'active',
    'disabled',
    'reported',
    'pending_review'
);

-- Create stories table
CREATE TABLE story (
    id SERIAL PRIMARY KEY,
    poi_id INTEGER NOT NULL REFERENCES poi(id) ON DELETE CASCADE,
    language VARCHAR(5) NOT NULL DEFAULT 'en',
    text TEXT NOT NULL,
    audio_url TEXT,
    duration_sec SMALLINT,
    layer_type story_layer_type NOT NULL DEFAULT 'general',
    order_index SMALLINT NOT NULL DEFAULT 0,
    is_inflation BOOLEAN NOT NULL DEFAULT false,
    confidence SMALLINT NOT NULL DEFAULT 80,
    sources JSONB DEFAULT '[]',
    status story_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create composite index on (poi_id, language, status) for filtered queries
CREATE INDEX idx_story_poi_language_status ON story (poi_id, language, status);
