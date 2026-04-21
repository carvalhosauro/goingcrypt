package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type Hasher struct {
	mock.Mock
}

func (m *Hasher) Hash(ctx context.Context, password string) (string, error) {
	args := m.Called(ctx, password)
	return args.String(0), args.Error(1)
}

func (m *Hasher) Verify(ctx context.Context, password, hash string) (bool, error) {
	args := m.Called(ctx, password, hash)
	return args.Bool(0), args.Error(1)
}
