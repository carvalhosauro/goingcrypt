package adapters

import (
	"context"
	"fmt"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TOTPAdapter struct{}

func NewTOTPAdapter() *TOTPAdapter {
	return &TOTPAdapter{}
}

func (a *TOTPAdapter) GenerateSecret(_ context.Context) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "goingcrypt",
		AccountName: "setup",
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", fmt.Errorf("generating TOTP key: %w", err)
	}
	return key.Secret(), nil
}

func (a *TOTPAdapter) GenerateProvisioningURI(_ context.Context, secret, username, issuer string) string {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: username,
		Secret:      []byte(secret),
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return ""
	}
	return key.URL()
}

func (a *TOTPAdapter) Validate(_ context.Context, secret, code string) bool {
	return totp.Validate(code, secret)
}
