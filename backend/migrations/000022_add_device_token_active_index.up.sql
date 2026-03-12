CREATE INDEX IF NOT EXISTS idx_device_tokens_active
    ON device_tokens (user_id, updated_at DESC)
    WHERE is_active = true;
