package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/google/uuid"
)

const refreshTokenTTL = 30 * 24 * time.Hour

type AuthService struct {
	userRepo     repository.UserRepository
	tokenRepo    repository.RefreshTokenRepository
	transactor   repository.Transactor
	generator    ports.Generator
	hasher       ports.Hasher
	tokenManager ports.TokenManager
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	transactor repository.Transactor,
	generator ports.Generator,
	hasher ports.Hasher,
	tokenManager ports.TokenManager,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		transactor:   transactor,
		generator:    generator,
		hasher:       hasher,
		tokenManager: tokenManager,
	}
}

func (s *AuthService) SignUp(ctx context.Context, in services.SignUpInput) (services.SignUpOutput, error) {
	existing, err := s.userRepo.GetByUsername(ctx, in.Username)
	if err != nil {
		return services.SignUpOutput{}, fmt.Errorf("checking username: %w", err)
	}
	if existing != nil {
		return services.SignUpOutput{}, domain.ErrUsernameTaken
	}

	id, err := s.generator.GenerateUUID(ctx)
	if err != nil {
		return services.SignUpOutput{}, fmt.Errorf("generating uuid: %w", err)
	}

	hashedPassword, err := s.hasher.Hash(ctx, in.Password)
	if err != nil {
		return services.SignUpOutput{}, fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		ID:       id,
		Username: in.Username,
		Password: hashedPassword,
		Role:     domain.UserRoleUser,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return services.SignUpOutput{}, fmt.Errorf("creating user: %w", err)
	}

	accessToken, rawRefresh, err := s.issueTokenPair(ctx, user.ID, user.Role, in.DeviceName, in.IPAddress, in.UserAgent)
	if err != nil {
		return services.SignUpOutput{}, err
	}

	return services.SignUpOutput{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, in services.LoginInput) (services.LoginOutput, error) {
	user, err := s.userRepo.GetByUsername(ctx, in.Username)
	if err != nil {
		return services.LoginOutput{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil || user.IsDeleted() {
		return services.LoginOutput{}, domain.ErrInvalidCredentials
	}

	ok, err := s.hasher.Verify(ctx, in.Password, user.Password)
	if err != nil {
		return services.LoginOutput{}, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return services.LoginOutput{}, domain.ErrInvalidCredentials
	}

	accessToken, rawRefresh, err := s.issueTokenPair(ctx, user.ID, user.Role, in.DeviceName, in.IPAddress, in.UserAgent)
	if err != nil {
		return services.LoginOutput{}, err
	}

	return services.LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// RefreshTokens rotates the refresh token: old one is revoked and replaced by a new pair.
func (s *AuthService) RefreshTokens(ctx context.Context, in services.RefreshTokensInput) (services.RefreshTokensOutput, error) {
	hash := hashToken(in.RefreshToken)

	stored, err := s.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		return services.RefreshTokensOutput{}, fmt.Errorf("fetching refresh token: %w", err)
	}
	if stored == nil || !stored.IsValid() {
		return services.RefreshTokensOutput{}, domain.ErrInvalidRefreshToken
	}

	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return services.RefreshTokensOutput{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil {
		return services.RefreshTokensOutput{}, domain.ErrUserNotFound
	}

	var accessToken, rawRefresh string
	if err := s.transactor.RunInTx(ctx, func(txCtx context.Context) error {
		newID, err := s.generator.GenerateUUID(txCtx)
		if err != nil {
			return fmt.Errorf("generating uuid: %w", err)
		}

		var newRawRefresh string
		accessToken, newRawRefresh, err = s.issueTokenPairInTx(txCtx, stored.UserID, user.Role, in.DeviceName, in.IPAddress, in.UserAgent)
		if err != nil {
			return err
		}
		rawRefresh = newRawRefresh

		stored.ReplacedBy = &newID
		stored.Revoke()
		if err := s.tokenRepo.Update(txCtx, stored); err != nil {
			return fmt.Errorf("revoking old refresh token: %w", err)
		}
		return nil
	}); err != nil {
		return services.RefreshTokensOutput{}, err
	}

	return services.RefreshTokensOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, in services.LogoutInput) error {
	hash := hashToken(in.RefreshToken)

	stored, err := s.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("fetching refresh token: %w", err)
	}
	if stored == nil || !stored.IsValid() {
		return domain.ErrInvalidRefreshToken
	}

	stored.Revoke()
	if err := s.tokenRepo.Update(ctx, stored); err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}

	return nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID uuid.UUID, role domain.UserRole, deviceName, ipAddress, userAgent string) (string, string, error) {
	return s.issueTokenPairInTx(ctx, userID, role, deviceName, ipAddress, userAgent)
}

func (s *AuthService) issueTokenPairInTx(ctx context.Context, userID uuid.UUID, role domain.UserRole, deviceName, ipAddress, userAgent string) (string, string, error) {
	claims := ports.TokenClaims{
		UserID: userID,
		Role:   role,
	}
	accessToken, err := s.tokenManager.GenerateAccessToken(ctx, claims)
	if err != nil {
		return "", "", fmt.Errorf("generating access token: %w", err)
	}

	rawRefresh, err := s.storeRefreshToken(ctx, userID, deviceName, ipAddress, userAgent)
	if err != nil {
		return "", "", err
	}

	return accessToken, rawRefresh, nil
}

func (s *AuthService) storeRefreshToken(ctx context.Context, userID uuid.UUID, deviceName, ipAddress, userAgent string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating refresh token bytes: %w", err)
	}
	rawHex := hex.EncodeToString(raw)

	id, err := s.generator.GenerateUUID(ctx)
	if err != nil {
		return "", fmt.Errorf("generating uuid: %w", err)
	}

	now := time.Now()
	token := &domain.RefreshToken{
		ID:         id,
		UserID:     userID,
		TokenHash:  hashToken(rawHex),
		DeviceName: deviceName,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		IssuedAt:   now,
		ExpiresAt:  now.Add(refreshTokenTTL),
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return "", fmt.Errorf("storing refresh token: %w", err)
	}

	return rawHex, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
