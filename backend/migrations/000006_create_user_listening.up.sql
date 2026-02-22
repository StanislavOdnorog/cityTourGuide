-- Create user_listening table
CREATE TABLE user_listening (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    story_id INTEGER NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    listened_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed BOOLEAN NOT NULL DEFAULT false,
    location GEOGRAPHY(POINT, 4326)
);

-- Composite index for deduplication and lookup
CREATE INDEX idx_user_listening_user_story ON user_listening (user_id, story_id);

-- Index for user history queries
CREATE INDEX idx_user_listening_user_listened ON user_listening (user_id, listened_at DESC);
