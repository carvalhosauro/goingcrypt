ALTER TABLE users
    ADD COLUMN mfa_enabled BOOLEAN DEFAULT FALSE,
    ADD COLUMN mfa_secret TEXT,
    ADD COLUMN recovery_codes TEXT[] NOT NULL DEFAULT '{}';
