CREATE TABLE link_access_logs (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    link_id    UUID UNIQUE NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    opened_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_access_link_id ON link_access_logs(link_id);
