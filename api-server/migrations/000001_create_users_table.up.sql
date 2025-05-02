CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id            UUID      PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         TEXT      NOT NULL UNIQUE,
    first_name    TEXT      NOT NULL,
    last_name     TEXT      NOT NULL,
    password_hash TEXT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
