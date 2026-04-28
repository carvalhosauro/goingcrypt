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

type AuthService interface {
	SignUp(ctx context.Context, in SignUpInput) (SignUpOutput, error)
	Login(ctx context.Context, in LoginInput) (LoginOutput, error)
	RefreshTokens(ctx context.Context, in RefreshTokensInput) (RefreshTokensOutput, error)
	Logout(ctx context.Context, in LogoutInput) error
}
