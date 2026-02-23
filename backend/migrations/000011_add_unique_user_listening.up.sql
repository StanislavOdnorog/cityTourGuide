-- Replace non-unique index with unique constraint for UPSERT support
DROP INDEX IF EXISTS idx_user_listening_user_story;
CREATE UNIQUE INDEX idx_user_listening_user_story ON user_listening (user_id, story_id);
