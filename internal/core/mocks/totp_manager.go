package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type TOTPManager struct {
	mock.Mock
}

func (m *TOTPManager) GenerateSecret(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *TOTPManager) GenerateProvisioningURI(ctx context.Context, secret, username, issuer string) string {
	return m.Called(ctx, secret, username, issuer).String(0)
}

func (m *TOTPManager) Validate(ctx context.Context, secret, code string) bool {
	return m.Called(ctx, secret, code).Bool(0)
}
