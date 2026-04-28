package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrLinkAlreadyOpened  = errors.New("link already opened")
	ErrLinkAlreadyExpired = errors.New("link already expired")
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

func (l *Link) CanAccess() bool {
	return l.Status == StatusWaiting && !l.IsExpired()
}

func (l *Link) IsExpired() bool {
	return l.ExpiresAt != nil && l.ExpiresAt.Before(time.Now())
}

func (l *Link) Open() (string, error) {
	switch l.Status {
	case StatusOpened:
		return "", ErrLinkAlreadyOpened
	case StatusExpired:
		return "", ErrLinkAlreadyExpired
	}
	if l.IsExpired() {
		return "", ErrLinkAlreadyExpired
	}

	text := l.CipheredText
	l.Status = StatusOpened
	l.CipheredText = ""

	return text, nil
}

func (l *Link) Invalidate() error {
	if l.Status == StatusOpened {
		return ErrLinkAlreadyOpened
	}
	l.Status = StatusExpired
	return nil
}

type LinkAccessLog struct {
	ID        uuid.UUID `db:"id"`
	LinkID    uuid.UUID `db:"link_id"`
	IPAddress string    `db:"ip_address"`
	UserAgent string    `db:"user_agent"`
	OpenedAt  time.Time `db:"opened_at"`
}

type LinkAccessEntry struct {
	ID        uuid.UUID `db:"id"`
	LinkID    uuid.UUID `db:"link_id"`
	Slug      string    `db:"slug"`
	IPAddress string    `db:"ip_address"`
	UserAgent string    `db:"user_agent"`
	OpenedAt  time.Time `db:"opened_at"`
}
