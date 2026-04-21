package adapters

import (
	"encoding/base32"

	"github.com/google/uuid"
)

type Generator struct {}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateUUID() (uuid.UUID, error) {
	return uuid.NewV7()
}

func (g *Generator) GenerateSlug(text string) (string, error) {
	return base32.StdEncoding.EncodeToString([]byte(text)), nil
}