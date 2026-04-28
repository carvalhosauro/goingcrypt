package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) conn(ctx context.Context) dbConn {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return r.db
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (id, username, password, role)
		VALUES (:id, :username, :password, :role)
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, user)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const query = `
		SELECT id, username, password, role, created_at, deleted_at
		FROM users
		WHERE id = $1
	`
	var user domain.User
	if err := r.conn(ctx).GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	const query = `
		SELECT id, username, password, role, created_at, deleted_at
		FROM users
		WHERE username = $1
	`
	var user domain.User
	if err := r.conn(ctx).GetContext(ctx, &user, query, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	const query = `
		UPDATE users
		SET password   = :password,
		    role        = :role,
		    deleted_at  = :deleted_at
		WHERE id = :id
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, user)
	return err
}
