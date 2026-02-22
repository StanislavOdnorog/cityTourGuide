-- Create ENUM type for auth provider
CREATE TYPE auth_provider AS ENUM (
    'email',
    'google',
    'apple'
);

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,
    name VARCHAR(255),
    auth_provider auth_provider NOT NULL DEFAULT 'email',
    language_pref VARCHAR(5) NOT NULL DEFAULT 'en',
    is_anonymous BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index on email for login lookups
CREATE INDEX idx_users_email ON users (email) WHERE email IS NOT NULL;
