package domain

import "errors"

var (
	ErrLinkNotFound = errors.New("link not found or expired")

	ErrUserNotFound        = errors.New("user not found")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUsernameTaken       = errors.New("username already taken")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)
