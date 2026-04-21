package ports

import (
	"context"

	"github.com/google/uuid"
)

type Generator interface {
	GenerateUUID(ctx context.Context) (uuid.UUID, error)
	GenerateSlug(ctx context.Context, text string) (string, error)
}