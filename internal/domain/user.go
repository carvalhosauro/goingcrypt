package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID      `db:"id"`
	Username   string         `db:"username"`
	Password   string         `db:"password"`
	MfaEnabled bool           `db:"mfa_enabled"`
	MfaSecret  sql.NullString `db:"mfa_secret"`
	CreatedAt  time.Time      `db:"created_at"`
	DeletedAt  *time.Time     `db:"deleted_at"`
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

func (u *User) Delete() {
	t := time.Now()
	u.DeletedAt = &t
}

func (u *User) UnDelete() {
	u.DeletedAt = nil
}

type RefreshToken struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TokenHash  string     `db:"token_hash"`
	DeviceName string     `db:"device_name"`
	IPAddress  string     `db:"ip_address"`
	UserAgent  string     `db:"user_agent"`
	IssuedAt   time.Time  `db:"issued_at"`
	ExpiresAt  time.Time  `db:"expires_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
	ReplacedBy *uuid.UUID `db:"replaced_by"`
}

func (r *RefreshToken) IsValid() bool {
	return r.RevokedAt == nil && r.ExpiresAt.After(time.Now())
}

func (r *RefreshToken) Revoke() {
	t := time.Now()
	r.RevokedAt = &t
}
