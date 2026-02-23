CREATE TYPE push_notification_type AS ENUM ('geo', 'content');

CREATE TABLE push_notifications (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_token_id INT NOT NULL REFERENCES device_tokens(id) ON DELETE CASCADE,
    type push_notification_type NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    data JSONB,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_notifications_user_type_date ON push_notifications(user_id, type, created_at);
