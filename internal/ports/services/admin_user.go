package services

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type GrantAdminInput struct {
	TargetUserID uuid.UUID
}

type AdminListUsersInput struct {
	Limit  int
	Offset int
}

type UserSummary struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Banned    bool      `json:"banned"`
}

type AdminListUsersOutput struct {
	Users []UserSummary `json:"users"`
	Total int           `json:"total"`
}

type AdminUserService interface {
	GrantAdmin(ctx context.Context, in GrantAdminInput) error
	ListUsers(ctx context.Context, in AdminListUsersInput) (AdminListUsersOutput, error)
}
