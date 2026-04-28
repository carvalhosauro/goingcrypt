package mocks

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type TokenManager struct {
	mock.Mock
}

func (m *TokenManager) GenerateAccessToken(ctx context.Context, userID uuid.UUID, role domain.UserRole) (string, error) {
	args := m.Called(ctx, userID, role)
	return args.String(0), args.Error(1)
}

func (m *TokenManager) ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, domain.UserRole, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(uuid.UUID), args.Get(1).(domain.UserRole), args.Error(2)
}
