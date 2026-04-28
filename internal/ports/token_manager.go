package ports

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/google/uuid"
)

type TokenClaims struct {
	UserID uuid.UUID
	Role   domain.UserRole
}

type TokenManager interface {
	GenerateAccessToken(ctx context.Context, claims TokenClaims) (string, error)
	ValidateAccessToken(ctx context.Context, token string) (TokenClaims, error)
}
