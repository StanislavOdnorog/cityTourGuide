-- Revert to non-unique index
DROP INDEX IF EXISTS idx_user_listening_user_story;
CREATE INDEX idx_user_listening_user_story ON user_listening (user_id, story_id);
