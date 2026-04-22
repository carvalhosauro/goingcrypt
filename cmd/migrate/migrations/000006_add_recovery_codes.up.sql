ALTER TABLE users
    ADD COLUMN recovery_codes TEXT[] NOT NULL DEFAULT '{}';
