package mocks

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type RefreshTokenRepository struct {
	mock.Mock
}

func (m *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	return m.Called(ctx, token).Error(0)
}

func (m *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *RefreshTokenRepository) Update(ctx context.Context, token *domain.RefreshToken) error {
	return m.Called(ctx, token).Error(0)
}

func (m *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
