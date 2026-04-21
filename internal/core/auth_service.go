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
	totp         ports.TOTPManager
	issuer       string
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	transactor repository.Transactor,
	generator ports.Generator,
	hasher ports.Hasher,
	tokenManager ports.TokenManager,
	totp ports.TOTPManager,
	issuer string,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		transactor:   transactor,
		generator:    generator,
		hasher:       hasher,
		tokenManager: tokenManager,
		totp:         totp,
		issuer:       issuer,
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
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return services.SignUpOutput{}, fmt.Errorf("creating user: %w", err)
	}

	accessToken, rawRefresh, err := s.issueTokenPair(ctx, user.ID, in.DeviceName, in.IPAddress, in.UserAgent)
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

	if user.MfaEnabled {
		return services.LoginOutput{MFARequired: true, UserID: user.ID}, nil
	}

	accessToken, rawRefresh, err := s.issueTokenPair(ctx, user.ID, in.DeviceName, in.IPAddress, in.UserAgent)
	if err != nil {
		return services.LoginOutput{}, err
	}

	return services.LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// LoginWithMFA re-validates credentials before accepting the TOTP code
// to prevent MFA bypass via direct calls to this endpoint.
func (s *AuthService) LoginWithMFA(ctx context.Context, in services.LoginWithMFAInput) (services.LoginWithMFAOutput, error) {
	user, err := s.userRepo.GetByUsername(ctx, in.Username)
	if err != nil {
		return services.LoginWithMFAOutput{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil || user.IsDeleted() {
		return services.LoginWithMFAOutput{}, domain.ErrInvalidCredentials
	}

	ok, err := s.hasher.Verify(ctx, in.Password, user.Password)
	if err != nil {
		return services.LoginWithMFAOutput{}, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return services.LoginWithMFAOutput{}, domain.ErrInvalidCredentials
	}

	if !user.MfaEnabled || !user.MfaSecret.Valid {
		return services.LoginWithMFAOutput{}, domain.ErrMFANotEnabled
	}

	if !s.totp.Validate(ctx, user.MfaSecret.String, in.Code) {
		return services.LoginWithMFAOutput{}, domain.ErrInvalidMFACode
	}

	accessToken, rawRefresh, err := s.issueTokenPair(ctx, user.ID, in.DeviceName, in.IPAddress, in.UserAgent)
	if err != nil {
		return services.LoginWithMFAOutput{}, err
	}

	return services.LoginWithMFAOutput{
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

	var accessToken, rawRefresh string
	if err := s.transactor.RunInTx(ctx, func(txCtx context.Context) error {
		newID, err := s.generator.GenerateUUID(txCtx)
		if err != nil {
			return fmt.Errorf("generating uuid: %w", err)
		}

		var newRawRefresh string
		accessToken, newRawRefresh, err = s.issueTokenPairInTx(txCtx, stored.UserID, in.DeviceName, in.IPAddress, in.UserAgent)
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

// EnableMFA generates a TOTP secret and provisioning URI without persisting them.
// The user must confirm with ConfirmMFA before MFA is activated.
func (s *AuthService) EnableMFA(ctx context.Context, in services.EnableMFAInput) (services.EnableMFAOutput, error) {
	user, err := s.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return services.EnableMFAOutput{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil {
		return services.EnableMFAOutput{}, domain.ErrUserNotFound
	}
	if user.MfaEnabled {
		return services.EnableMFAOutput{}, domain.ErrMFAAlreadyEnabled
	}

	secret, err := s.totp.GenerateSecret(ctx)
	if err != nil {
		return services.EnableMFAOutput{}, fmt.Errorf("generating TOTP secret: %w", err)
	}

	uri := s.totp.GenerateProvisioningURI(ctx, secret, user.Username, s.issuer)

	return services.EnableMFAOutput{
		Secret:          secret,
		ProvisioningURI: uri,
	}, nil
}

// ConfirmMFA validates the TOTP code against the provided secret and persists MFA on the user.
func (s *AuthService) ConfirmMFA(ctx context.Context, in services.ConfirmMFAInput) error {
	user, err := s.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}
	if user == nil {
		return domain.ErrUserNotFound
	}
	if user.MfaEnabled {
		return domain.ErrMFAAlreadyEnabled
	}

	if !s.totp.Validate(ctx, in.Secret, in.Code) {
		return domain.ErrInvalidMFACode
	}

	user.MfaEnabled = true
	user.MfaSecret.String = in.Secret
	user.MfaSecret.Valid = true

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("enabling MFA: %w", err)
	}

	return nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID uuid.UUID, deviceName, ipAddress, userAgent string) (string, string, error) {
	return s.issueTokenPairInTx(ctx, userID, deviceName, ipAddress, userAgent)
}

func (s *AuthService) issueTokenPairInTx(ctx context.Context, userID uuid.UUID, deviceName, ipAddress, userAgent string) (string, string, error) {
	accessToken, err := s.tokenManager.GenerateAccessToken(ctx, userID)
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
