package domain

import (
	"time"

	"github.com/google/uuid"
)

type LinkStatus string

const (
    StatusWaiting LinkStatus = "WAITING"
    StatusOpened  LinkStatus = "OPENED"
    StatusExpired LinkStatus = "EXPIRED"
)

type Link struct {
    ID           uuid.UUID  `db:"id"`
    Slug         string     `db:"slug"`
    HashedKey    string     `db:"hashed_key"`
    CipheredText string     `db:"ciphered_text"`
    CreatedAt    time.Time  `db:"created_at"`
    ExpiresAt    *time.Time `db:"expires_at"`
    Status       LinkStatus `db:"status"`
    CreatedBy    *uuid.UUID `db:"created_by"`
}

type LinkAccessLogs struct {
    ID        uuid.UUID `db:"id"`
    LinkID    uuid.UUID `db:"link_id"`
    IPAddress string    `db:"ip_address"`
    UserAgent string    `db:"user_agent"`
    OpenedAt  time.Time `db:"opened_at"`
}