package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/google/uuid"
)

const refreshTokenTTL = 30 * 24 * time.Hour

// recoveryCodeAlphabet and length define the shape of each recovery code.
// Format: XXXX-XXXX-XXXX  (3 groups of 4 uppercase alphanumerics).
const (
	recoveryCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/I/1 for readability
	recoveryCodeGroupLen = 4
	recoveryCodeGroups   = 3
	recoveryCodeCount    = 8
)

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

	if user.MfaEnabled {
		return services.LoginOutput{MFARequired: true, UserID: user.ID}, nil
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

	accessToken, rawRefresh, err := s.issueTokenPair(ctx, user.ID, user.Role, in.DeviceName, in.IPAddress, in.UserAgent)
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

// ConfirmMFA validates the TOTP code against the provided secret, persists MFA on the
// user and generates a one-time set of recovery codes. The plaintext codes are returned
// exactly once — they are never stored in plain text and cannot be retrieved again.
func (s *AuthService) ConfirmMFA(ctx context.Context, in services.ConfirmMFAInput) (services.ConfirmMFAOutput, error) {
	user, err := s.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return services.ConfirmMFAOutput{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil {
		return services.ConfirmMFAOutput{}, domain.ErrUserNotFound
	}
	if user.MfaEnabled {
		return services.ConfirmMFAOutput{}, domain.ErrMFAAlreadyEnabled
	}

	if !s.totp.Validate(ctx, in.Secret, in.Code) {
		return services.ConfirmMFAOutput{}, domain.ErrInvalidMFACode
	}

	plainCodes := make([]string, recoveryCodeCount)
	hashedCodes := make([]string, recoveryCodeCount)
	for i := range plainCodes {
		code, err := generateRecoveryCode()
		if err != nil {
			return services.ConfirmMFAOutput{}, fmt.Errorf("generating recovery code: %w", err)
		}
		plainCodes[i] = code
		hashedCodes[i] = hashToken(code)
	}

	user.MfaEnabled = true
	user.MfaSecret.String = in.Secret
	user.MfaSecret.Valid = true
	user.RecoveryCodes = hashedCodes

	if err := s.userRepo.Update(ctx, user); err != nil {
		return services.ConfirmMFAOutput{}, fmt.Errorf("enabling MFA: %w", err)
	}

	return services.ConfirmMFAOutput{RecoveryCodes: plainCodes}, nil
}

// RecoveryConfirm validates a one-time recovery code, resets the password,
// disables MFA and issues a fresh token pair.
func (s *AuthService) RecoveryConfirm(ctx context.Context, in services.RecoveryConfirmInput) (services.RecoveryConfirmOutput, error) {
	user, err := s.userRepo.GetByUsername(ctx, in.Username)
	if err != nil {
		return services.RecoveryConfirmOutput{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil || user.IsDeleted() {
		return services.RecoveryConfirmOutput{}, domain.ErrUserNotFound
	}

	// Find and consume the matching recovery code (constant-time comparison via hash).
	incomingHash := hashToken(in.RecoveryCode)
	matchIndex := -1
	for i, h := range user.RecoveryCodes {
		if h == incomingHash {
			matchIndex = i
			break
		}
	}
	if matchIndex == -1 {
		return services.RecoveryConfirmOutput{}, domain.ErrInvalidRecoveryCode
	}

	// Remove the used code (single-use).
	remaining := make([]string, 0, len(user.RecoveryCodes)-1)
	remaining = append(remaining, user.RecoveryCodes[:matchIndex]...)
	remaining = append(remaining, user.RecoveryCodes[matchIndex+1:]...)

	newHash, err := s.hasher.Hash(ctx, in.NewPassword)
	if err != nil {
		return services.RecoveryConfirmOutput{}, fmt.Errorf("hashing new password: %w", err)
	}

	user.Password = newHash
	user.MfaEnabled = false
	user.MfaSecret.Valid = false
	user.MfaSecret.String = ""
	user.RecoveryCodes = remaining

	var accessToken, rawRefresh string
	if err := s.transactor.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return fmt.Errorf("updating user: %w", err)
		}
		at, rt, err := s.issueTokenPairInTx(txCtx, user.ID, user.Role, in.DeviceName, in.IPAddress, in.UserAgent)
		if err != nil {
			return err
		}
		accessToken, rawRefresh = at, rt
		return nil
	}); err != nil {
		return services.RecoveryConfirmOutput{}, err
	}

	return services.RecoveryConfirmOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
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

// generateRecoveryCode creates one recovery code in the format XXXX-XXXX-XXXX.
func generateRecoveryCode() (string, error) {
	n := big.NewInt(int64(len(recoveryCodeAlphabet)))
	groups := make([]byte, recoveryCodeGroups*recoveryCodeGroupLen)
	for i := range groups {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		groups[i] = recoveryCodeAlphabet[idx.Int64()]
	}

	var code string
	for g := 0; g < recoveryCodeGroups; g++ {
		if g > 0 {
			code += "-"
		}
		code += string(groups[g*recoveryCodeGroupLen : (g+1)*recoveryCodeGroupLen])
	}
	return code, nil
}
