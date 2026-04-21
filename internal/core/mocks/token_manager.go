package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type TokenManager struct {
	mock.Mock
}

func (m *TokenManager) GenerateAccessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *TokenManager) ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
