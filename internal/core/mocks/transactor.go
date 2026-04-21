package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type Transactor struct {
	mock.Mock
}

func (m *Transactor) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) == nil {
		return fn(ctx)
	}
	return args.Error(0)
}
