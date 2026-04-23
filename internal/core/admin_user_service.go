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
