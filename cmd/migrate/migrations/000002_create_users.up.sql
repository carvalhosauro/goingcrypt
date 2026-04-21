DO $$ BEGIN
    CREATE TYPE link_status AS ENUM ('WAITING', 'OPENED', 'EXPIRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    username    VARCHAR(50) UNIQUE NOT NULL,
    password    TEXT NOT NULL,
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret  TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE
);
