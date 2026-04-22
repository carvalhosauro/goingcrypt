CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  CHAR(64) NOT NULL UNIQUE,
    device_name VARCHAR(100),
    ip_address  INET,
    user_agent  TEXT,
    issued_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at  TIMESTAMP WITH TIME ZONE,
    replaced_by UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL
);

CREATE INDEX idx_rt_user_id         ON refresh_tokens(user_id);
CREATE INDEX idx_rt_token_hash      ON refresh_tokens(token_hash);
CREATE INDEX idx_rt_active_sessions ON refresh_tokens(user_id)
    WHERE revoked_at IS NULL;
