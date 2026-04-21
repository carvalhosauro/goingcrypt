package ports

import (
	"context"

	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error)
}
