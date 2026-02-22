-- Create ENUM type for purchase type
CREATE TYPE purchase_type AS ENUM (
    'city_pack',
    'subscription',
    'lifetime'
);

-- Create purchases table
CREATE TABLE purchase (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type purchase_type NOT NULL,
    city_id INTEGER REFERENCES cities(id) ON DELETE SET NULL,
    platform VARCHAR(10) NOT NULL,
    transaction_id TEXT,
    price DECIMAL(10, 2) NOT NULL,
    is_ltd BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for user purchase lookups
CREATE INDEX idx_purchase_user ON purchase (user_id);

-- Index for transaction deduplication
CREATE UNIQUE INDEX idx_purchase_transaction ON purchase (transaction_id) WHERE transaction_id IS NOT NULL;
