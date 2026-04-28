package core

import (
	"context"
	"fmt"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
)

type AdminUserService struct {
	userRepo repository.UserRepository
}

func NewAdminUserService(userRepo repository.UserRepository) *AdminUserService {
	return &AdminUserService{userRepo: userRepo}
}

func (s *AdminUserService) GrantAdmin(ctx context.Context, in services.GrantAdminInput) error {
	user, err := s.userRepo.GetByID(ctx, in.TargetUserID)
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}
	if user == nil {
		return domain.ErrUserNotFound
	}
	user.Role = domain.UserRoleAdmin
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("updating user role: %w", err)
	}
	return nil
}

func (s *AdminUserService) ListUsers(ctx context.Context, in services.AdminListUsersInput) (services.AdminListUsersOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}

	users, err := s.userRepo.List(ctx, limit, in.Offset)
	if err != nil {
		return services.AdminListUsersOutput{}, fmt.Errorf("listing users: %w", err)
	}

	summaries := make([]services.UserSummary, len(users))
	for i, u := range users {
		summaries[i] = services.UserSummary{
			ID:        u.ID.String(),
			Username:  u.Username,
			Role:      string(u.Role),
			CreatedAt: u.CreatedAt,
			Banned:    u.DeletedAt != nil,
		}
	}

	return services.AdminListUsersOutput{
		Users: summaries,
		Total: len(summaries),
	}, nil
}
