ALTER TABLE users ADD COLUMN provider_id VARCHAR(255);

CREATE UNIQUE INDEX idx_users_provider ON users (auth_provider, provider_id) WHERE provider_id IS NOT NULL;
