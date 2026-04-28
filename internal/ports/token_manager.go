package ports

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(ctx context.Context, userID uuid.UUID, role domain.UserRole) (string, error)
	ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, domain.UserRole, error)
}
