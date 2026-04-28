package services

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AccessLinkInput struct {
	Slug      string
	Key       string
	IPAddress string
	UserAgent string
}

type AccessLinkOutput struct {
	CipheredText string
}

type CreateLinkInput struct {
	Key          string
	CipheredText string
	ExpiresIn    *time.Duration
	CreatedBy    *uuid.UUID
}

type CreateLinkOutput struct {
	Slug string
}

type DeleteLinkInput struct {
	Slug   string
	UserID uuid.UUID
}

type ListMyLinksInput struct {
	UserID uuid.UUID
	Limit  int
	Offset int
}

type MyLinkSummary struct {
	Slug      string     `json:"slug"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type ListMyLinksOutput struct {
	Links []MyLinkSummary `json:"links"`
}

type LinkService interface {
	AccessLink(ctx context.Context, in AccessLinkInput) (AccessLinkOutput, error)
	CreateLink(ctx context.Context, in CreateLinkInput) (CreateLinkOutput, error)
	DeleteLink(ctx context.Context, in DeleteLinkInput) error
	ListMyLinks(ctx context.Context, in ListMyLinksInput) (ListMyLinksOutput, error)
}

// ─── Admin types ────────────────────────────────────────────────────────────

type AdminListLinksInput struct {
	Limit  int
	Offset int
}

type AdminLinkSummary struct {
	Slug      string     `json:"slug"`
	Status    string     `json:"status"`
	CreatedBy *uuid.UUID `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type AdminListLinksOutput struct {
	Links  []AdminLinkSummary `json:"links"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type AdminGetLinkInput struct {
	ID string
}

type AdminLinkDetail struct {
	ID        uuid.UUID  `json:"id"`
	Slug      string     `json:"slug"`
	Status    string     `json:"status"`
	CreatedBy *uuid.UUID `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type AdminGetLinkOutput struct {
	Link AdminLinkDetail `json:"link"`
}

type AdminListAccessLogsInput struct {
	Limit  int
	Offset int
}

type AdminAccessLogEntry struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	OpenedAt  time.Time `json:"opened_at"`
}

type AdminListAccessLogsOutput struct {
	Logs  []AdminAccessLogEntry `json:"logs"`
	Total int                   `json:"total"`
}

type AdminLinkService interface {
	ListLinks(ctx context.Context, in AdminListLinksInput) (AdminListLinksOutput, error)
	GetLink(ctx context.Context, in AdminGetLinkInput) (AdminGetLinkOutput, error)
	ListAccessLogs(ctx context.Context, in AdminListAccessLogsInput) (AdminListAccessLogsOutput, error)
}
