package repository

import (
	"context"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/google/uuid"
)

type LinkOption func(*LinkFilter)

type LinkFilter struct {
	Status    *domain.LinkStatus
	CreatedBy *uuid.UUID
	ExpiresAt *time.Time

	Limit  int
	Offset int
}

func WithStatus(s domain.LinkStatus) LinkOption {
	return func(f *LinkFilter) {
		f.Status = &s
	}
}

func WithCreatedBy(userID uuid.UUID) LinkOption {
	return func(f *LinkFilter) {
		f.CreatedBy = &userID
	}
}

func WithExpiringBefore(t time.Time) LinkOption {
	return func(f *LinkFilter) {
		f.ExpiresAt = &t
	}
}

func WithPagination(limit, offset int) LinkOption {
	return func(f *LinkFilter) {
		f.Limit = limit
		f.Offset = offset
	}
}

func NewLinkFilter(opts ...LinkOption) *LinkFilter {
	f := &LinkFilter{
		Limit: 50,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

type LinkRepository interface {
	Create(ctx context.Context, link *domain.Link) error
	GetBySlug(ctx context.Context, slug string) (*domain.Link, error)
	Update(ctx context.Context, link *domain.Link) error
	InvalidateExpiredLinks(ctx context.Context) error
	Delete(ctx context.Context, slug string) error
	List(ctx context.Context, opts ...LinkOption) ([]domain.Link, error)
}