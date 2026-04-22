package adapters

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateUUID(_ context.Context) (uuid.UUID, error) {
	return uuid.NewV7()
}

// GenerateSlug encodes the UUID bytes as base64url (no padding) → exactly 22 chars,
// matching the VARCHAR(22) slug column in the schema.
func (g *Generator) GenerateSlug(_ context.Context, text string) (string, error) {
	id, err := uuid.Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing uuid for slug: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(id[:]), nil
}
