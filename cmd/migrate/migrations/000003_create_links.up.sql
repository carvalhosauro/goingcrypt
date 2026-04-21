CREATE TABLE links (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    slug          VARCHAR(22) NOT NULL UNIQUE,
    hashed_key    CHAR(64)    NOT NULL UNIQUE,
    ciphered_text TEXT        NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at    TIMESTAMP WITH TIME ZONE,
    status        link_status DEFAULT 'WAITING',
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_links_slug       ON links(slug);
CREATE INDEX idx_links_hashed_key ON links(hashed_key);
CREATE INDEX idx_links_status     ON links(status);
CREATE INDEX idx_links_created_by ON links(created_by);
