package mocks

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/stretchr/testify/mock"
)

type LinkRepository struct {
	mock.Mock
}

func (m *LinkRepository) Create(ctx context.Context, link *domain.Link) error {
	return m.Called(ctx, link).Error(0)
}

func (m *LinkRepository) GetBySlug(ctx context.Context, slug string) (*domain.Link, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Link), args.Error(1)
}

func (m *LinkRepository) Update(ctx context.Context, link *domain.Link) error {
	return m.Called(ctx, link).Error(0)
}

func (m *LinkRepository) CreateAccessLog(ctx context.Context, log *domain.LinkAccessLog) error {
	return m.Called(ctx, log).Error(0)
}

func (m *LinkRepository) InvalidateExpiredLinks(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *LinkRepository) Delete(ctx context.Context, slug string) error {
	return m.Called(ctx, slug).Error(0)
}

func (m *LinkRepository) List(ctx context.Context, opts ...repository.LinkOption) ([]domain.Link, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]domain.Link), args.Error(1)
}
