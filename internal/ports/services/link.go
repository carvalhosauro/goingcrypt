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

type LinkService interface {
	AccessLink(ctx context.Context, in AccessLinkInput) (AccessLinkOutput, error)
	CreateLink(ctx context.Context, in CreateLinkInput) (CreateLinkOutput, error)
	DeleteLink(ctx context.Context, in DeleteLinkInput) error
}
