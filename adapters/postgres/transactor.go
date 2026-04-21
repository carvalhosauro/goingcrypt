package postgres

import (
	"context"
	"fmt"

	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/jmoiron/sqlx"
)

type txKey struct{}

type transactor struct {
	db *sqlx.DB
}

func NewTransactor(db *sqlx.DB) repository.Transactor {
	return &transactor{db: db}
}

func (t *transactor) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
