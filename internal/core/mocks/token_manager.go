package mocks

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/stretchr/testify/mock"
)

type TokenManager struct {
	mock.Mock
}

func (m *TokenManager) GenerateAccessToken(ctx context.Context, claims ports.TokenClaims) (string, error) {
	args := m.Called(ctx, claims)
	return args.String(0), args.Error(1)
}

func (m *TokenManager) ValidateAccessToken(ctx context.Context, token string) (ports.TokenClaims, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(ports.TokenClaims), args.Error(1)
}
