package ports

import "context"

type TOTPManager interface {
	GenerateSecret(ctx context.Context) (string, error)
	GenerateProvisioningURI(ctx context.Context, secret, username, issuer string) string
	Validate(ctx context.Context, secret, code string) bool
}
