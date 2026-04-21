package adapters

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP recommended minimums)
const (
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 4
	argonSaltLen     = 16
	argonKeyLen      = 32
)

type Argon2idHasher struct{}

func NewArgon2idHasher() *Argon2idHasher {
	return &Argon2idHasher{}
}

func (h *Argon2idHasher) Hash(_ context.Context, password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func (h *Argon2idHasher) Verify(_ context.Context, password, encoded string) (bool, error) {
	salt, hash, err := decodeArgon2idHash(encoded)
	if err != nil {
		return false, fmt.Errorf("decoding hash: %w", err)
	}

	candidate := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)

	return subtle.ConstantTimeCompare(hash, candidate) == 1, nil
}

func decodeArgon2idHash(encoded string) (salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	// format: $argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>
	// split by $ gives: ["", "argon2id", "v=19", "m=...", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return nil, nil, fmt.Errorf("invalid hash format")
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, fmt.Errorf("decoding hash: %w", err)
	}

	return salt, hash, nil
}
