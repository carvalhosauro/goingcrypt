package repository

import (
	"context"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/google/uuid"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Update(ctx context.Context, token *domain.RefreshToken) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}
