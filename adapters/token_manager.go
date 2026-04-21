package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const accessTokenTTL = 15 * time.Minute

type jwtClaims struct {
	jwt.RegisteredClaims
}

type JWTTokenManager struct {
	secret []byte
	issuer string
}

func NewJWTTokenManager(secret []byte, issuer string) *JWTTokenManager {
	return &JWTTokenManager{secret: secret, issuer: issuer}
}

func (m *JWTTokenManager) GenerateAccessToken(_ context.Context, userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

func (m *JWTTokenManager) ValidateAccessToken(_ context.Context, tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())

	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing user id from token: %w", err)
	}

	return userID, nil
}
