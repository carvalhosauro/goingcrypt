package services

import (
	"context"

	"github.com/google/uuid"
)

type GrantAdminInput struct {
	TargetUserID uuid.UUID
}

type AdminUserService interface {
	GrantAdmin(ctx context.Context, in GrantAdminInput) error
}
