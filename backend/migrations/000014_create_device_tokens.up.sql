CREATE TABLE device_tokens (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform VARCHAR(10) NOT NULL CHECK (platform IN ('ios', 'android')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_device_tokens_token ON device_tokens(token);
CREATE INDEX idx_device_tokens_user_id ON device_tokens(user_id);
