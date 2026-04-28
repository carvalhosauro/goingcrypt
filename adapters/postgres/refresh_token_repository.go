package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type refreshTokenRepository struct {
	db *sqlx.DB
}

func NewRefreshTokenRepository(db *sqlx.DB) repository.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) conn(ctx context.Context) dbConn {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return r.db
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, device_name, ip_address, user_agent, issued_at, expires_at)
		VALUES (:id, :user_id, :token_hash, :device_name, :ip_address, :user_agent, :issued_at, :expires_at)
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, token)
	return err
}

func (r *refreshTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, device_name, ip_address, user_agent,
		       issued_at, expires_at, revoked_at, replaced_by
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var token domain.RefreshToken
	if err := r.conn(ctx).GetContext(ctx, &token, query, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepository) Update(ctx context.Context, token *domain.RefreshToken) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at  = :revoked_at,
		    replaced_by = :replaced_by
		WHERE id = :id
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, token)
	return err
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.conn(ctx).ExecContext(ctx, query, userID)
	return err
}

func (r *refreshTokenRepository) DeleteExpiredAndRevoked(ctx context.Context, olderThan time.Time) (int64, error) {
	const query = `
		DELETE FROM refresh_tokens
		WHERE (revoked_at IS NOT NULL OR expires_at < NOW())
		AND issued_at < $1
	`
	result, err := r.conn(ctx).ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
