package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/jmoiron/sqlx"
)

// dbConn é implementada tanto por *sqlx.DB quanto por *sqlx.Tx,
// permitindo que os métodos do repositório funcionem dentro ou fora de uma transação.
type dbConn interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

type linkRepository struct {
	db *sqlx.DB
}

func NewLinkRepository(db *sqlx.DB) repository.LinkRepository {
	return &linkRepository{db: db}
}

// conn retorna a transação ativa do contexto, ou o pool de conexões como fallback.
func (r *linkRepository) conn(ctx context.Context) dbConn {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return r.db
}

func (r *linkRepository) Create(ctx context.Context, link *domain.Link) error {
	const query = `
		INSERT INTO links (id, slug, hashed_key, ciphered_text, expires_at, status, created_by)
		VALUES (:id, :slug, :hashed_key, :ciphered_text, :expires_at, :status, :created_by)
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, link)
	return err
}

func (r *linkRepository) GetBySlug(ctx context.Context, slug string) (*domain.Link, error) {
	const query = `
		SELECT id, slug, hashed_key, ciphered_text, created_at, expires_at, status, created_by
		FROM links
		WHERE slug = $1
	`
	var link domain.Link
	err := r.conn(ctx).GetContext(ctx, &link, query, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &link, nil
}

func (r *linkRepository) Update(ctx context.Context, link *domain.Link) error {
	const query = `
		UPDATE links
		SET hashed_key    = :hashed_key,
		    ciphered_text = :ciphered_text,
		    expires_at    = :expires_at,
		    status        = :status
		WHERE id = :id
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, link)
	return err
}

func (r *linkRepository) CreateAccessLog(ctx context.Context, log *domain.LinkAccessLog) error {
	const query = `
		INSERT INTO link_access_logs (id, link_id, ip_address, user_agent, opened_at)
		VALUES (:id, :link_id, :ip_address, :user_agent, :opened_at)
	`
	_, err := r.conn(ctx).NamedExecContext(ctx, query, log)
	return err
}

func (r *linkRepository) InvalidateExpiredLinks(ctx context.Context) error {
	const query = `
		UPDATE links
		SET status = 'EXPIRED'
		WHERE status = 'WAITING'
		AND expires_at < NOW()
	`
	_, err := r.conn(ctx).ExecContext(ctx, query)
	return err
}

func (r *linkRepository) Delete(ctx context.Context, slug string) error {
	const query = `DELETE FROM links WHERE slug = $1`
	_, err := r.conn(ctx).ExecContext(ctx, query, slug)
	return err
}

func (r *linkRepository) List(ctx context.Context, opts ...repository.LinkOption) ([]domain.Link, error) {
	filter := repository.NewLinkFilter(opts...)

	query := `
		SELECT id, slug, hashed_key, ciphered_text, created_at, expires_at, status, created_by
		FROM links
		WHERE 1=1
	`
	args := make([]any, 0)
	argIdx := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.CreatedBy != nil {
		query += fmt.Sprintf(" AND created_by = $%d", argIdx)
		args = append(args, *filter.CreatedBy)
		argIdx++
	}

	if filter.ExpiresAt != nil {
		query += fmt.Sprintf(" AND expires_at < $%d", argIdx)
		args = append(args, *filter.ExpiresAt)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	var links []domain.Link
	if err := r.conn(ctx).SelectContext(ctx, &links, query, args...); err != nil {
		return nil, err
	}
	return links, nil
}

func (r *linkRepository) ListAccessLogs(ctx context.Context, limit, offset int) ([]domain.LinkAccessEntry, error) {
	const query = `
		SELECT lal.id, lal.link_id, l.slug, lal.ip_address, lal.user_agent, lal.opened_at
		FROM link_access_logs lal
		JOIN links l ON l.id = lal.link_id
		ORDER BY lal.opened_at DESC
		LIMIT $1 OFFSET $2
	`
	var entries []domain.LinkAccessEntry
	if err := r.conn(ctx).SelectContext(ctx, &entries, query, limit, offset); err != nil {
		return nil, err
	}
	return entries, nil
}
