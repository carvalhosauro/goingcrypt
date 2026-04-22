package services

import (
	"context"

	"github.com/google/uuid"
)

type SignUpInput struct {
	Username   string
	Password   string
	DeviceName string
	IPAddress  string
	UserAgent  string
}

type SignUpOutput struct {
	UserID       uuid.UUID
	AccessToken  string
	RefreshToken string
}

type LoginInput struct {
	Username   string
	Password   string
	DeviceName string
	IPAddress  string
	UserAgent  string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	MFARequired  bool
	UserID       uuid.UUID
}

// LoginWithMFAInput is used when MFA is required after Login.
// Credentials are re-validated to prevent MFA bypass.
type LoginWithMFAInput struct {
	Username   string
	Password   string
	Code       string
	DeviceName string
	IPAddress  string
	UserAgent  string
}

type LoginWithMFAOutput struct {
	AccessToken  string
	RefreshToken string
}

type RefreshTokensInput struct {
	RefreshToken string
	DeviceName   string
	IPAddress    string
	UserAgent    string
}

type RefreshTokensOutput struct {
	AccessToken  string
	RefreshToken string
}

type LogoutInput struct {
	RefreshToken string
}

type EnableMFAInput struct {
	UserID uuid.UUID
}

type EnableMFAOutput struct {
	Secret          string
	ProvisioningURI string
}

// ConfirmMFAInput finalizes MFA enrollment.
// Secret is provided by the client (returned from EnableMFA) because
// it hasn't been persisted yet — only confirmed codes trigger persistence.
type ConfirmMFAInput struct {
	UserID uuid.UUID
	Secret string
	Code   string
}

// ConfirmMFAOutput contains the one-time recovery codes generated when MFA
// is activated. The codes are shown exactly once and must be stored securely
// by the client — there is no way to retrieve them again.
type ConfirmMFAOutput struct {
	RecoveryCodes []string
}

// RecoveryConfirmInput finalizes recovery: validates a single-use recovery
// code, resets the password and disables MFA. New tokens are issued on success.
type RecoveryConfirmInput struct {
	Username     string
	RecoveryCode string
	NewPassword  string
	DeviceName   string
	IPAddress    string
	UserAgent    string
}

type RecoveryConfirmOutput struct {
	AccessToken  string
	RefreshToken string
}

type AuthService interface {
	SignUp(ctx context.Context, in SignUpInput) (SignUpOutput, error)
	Login(ctx context.Context, in LoginInput) (LoginOutput, error)
	LoginWithMFA(ctx context.Context, in LoginWithMFAInput) (LoginWithMFAOutput, error)
	RefreshTokens(ctx context.Context, in RefreshTokensInput) (RefreshTokensOutput, error)
	Logout(ctx context.Context, in LogoutInput) error
	EnableMFA(ctx context.Context, in EnableMFAInput) (EnableMFAOutput, error)
	ConfirmMFA(ctx context.Context, in ConfirmMFAInput) (ConfirmMFAOutput, error)
	RecoveryConfirm(ctx context.Context, in RecoveryConfirmInput) (RecoveryConfirmOutput, error)
}
