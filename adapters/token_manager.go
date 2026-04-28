package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const accessTokenTTL = 15 * time.Minute

type jwtClaims struct {
	jwt.RegisteredClaims
	Role domain.UserRole `json:"role"`
}

type JWTTokenManager struct {
	secret []byte
	issuer string
}

func NewJWTTokenManager(secret []byte, issuer string) *JWTTokenManager {
	return &JWTTokenManager{secret: secret, issuer: issuer}
}

func (m *JWTTokenManager) GenerateAccessToken(_ context.Context, claims ports.TokenClaims) (string, error) {
	now := time.Now()
	jClaims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
		Role: claims.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jClaims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

func (m *JWTTokenManager) ValidateAccessToken(_ context.Context, tokenStr string) (ports.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())

	if err != nil {
		return ports.TokenClaims{}, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return ports.TokenClaims{}, fmt.Errorf("invalid token claims")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return ports.TokenClaims{}, fmt.Errorf("parsing user id from token: %w", err)
	}

	return ports.TokenClaims{UserID: userID, Role: claims.Role}, nil
}
