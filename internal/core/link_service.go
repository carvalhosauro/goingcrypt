package core

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
)

type LinkService struct {
	linkRepo  repository.LinkRepository
	generator ports.Generator
}

func NewLinkService(linkRepo repository.LinkRepository, generator ports.Generator) *LinkService {
	return &LinkService{linkRepo: linkRepo, generator: generator}
}

func (s *LinkService) AccessLink(ctx context.Context, in services.AccessLinkInput) (services.AccessLinkOutput, error) {
	link, err := s.linkRepo.GetBySlug(ctx, in.Slug)
	if err != nil {
		return services.AccessLinkOutput{}, fmt.Errorf("fetching link: %w", err)
	}

	if link == nil || !link.CanAccess() {
		if link != nil && link.IsExpired() {
			if err := link.Invalidate(); err == nil {
				_ = s.linkRepo.Update(ctx, link)
			}
		}
		return services.AccessLinkOutput{}, domain.ErrLinkNotFound
	}

	hashedKey := sha256.Sum256([]byte(in.Key))
	hashedKeyHex := hex.EncodeToString(hashedKey[:])
	if subtle.ConstantTimeCompare([]byte(link.HashedKey), []byte(hashedKeyHex)) != 1 {
		return services.AccessLinkOutput{}, domain.ErrLinkNotFound
	}

	cipheredText, err := link.Open()
	if err != nil {
		return services.AccessLinkOutput{}, fmt.Errorf("opening link: %w", err)
	}

	if err := s.linkRepo.Update(ctx, link); err != nil {
		return services.AccessLinkOutput{}, fmt.Errorf("updating link status: %w", err)
	}

	linkID := link.ID
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logID, err := s.generator.GenerateUUID(bgCtx)
		if err != nil {
			slog.Error("failed to generate access log id", "err", err, "link_id", linkID)
			return
		}

		accessLog := &domain.LinkAccessLog{
			ID:        logID,
			LinkID:    linkID,
			IPAddress: in.IPAddress,
			UserAgent: in.UserAgent,
			OpenedAt:  time.Now(),
		}

		if err := s.linkRepo.CreateAccessLog(bgCtx, accessLog); err != nil {
			slog.Error("failed to create access log", "err", err, "link_id", linkID)
		}
	}()

	return services.AccessLinkOutput{CipheredText: cipheredText}, nil
}

func (s *LinkService) CreateLink(ctx context.Context, in services.CreateLinkInput) (services.CreateLinkOutput, error) {
	id, err := s.generator.GenerateUUID(ctx)
	if err != nil {
		return services.CreateLinkOutput{}, fmt.Errorf("generating uuid: %w", err)
	}

	slug, err := s.generator.GenerateSlug(ctx, id.String())
	if err != nil {
		return services.CreateLinkOutput{}, fmt.Errorf("generating slug: %w", err)
	}

	hashedKey := sha256.Sum256([]byte(in.Key))
	hashedKeyHex := hex.EncodeToString(hashedKey[:])
	var expiresAt *time.Time
	if in.ExpiresIn != nil {
		t := time.Now().Add(*in.ExpiresIn)
		expiresAt = &t
	}

	link := &domain.Link{
		ID:           id,
		Slug:         slug,
		HashedKey:    hashedKeyHex,
		CipheredText: in.CipheredText,
		ExpiresAt:    expiresAt,
		CreatedBy:    in.CreatedBy,
		Status:       domain.StatusWaiting,
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return services.CreateLinkOutput{}, fmt.Errorf("creating link: %w", err)
	}

	return services.CreateLinkOutput{
		Slug: link.Slug,
	}, nil
}

func (s *LinkService) DeleteLink(ctx context.Context, in services.DeleteLinkInput) error {
	link, err := s.linkRepo.GetBySlug(ctx, in.Slug)
	if err != nil {
		return fmt.Errorf("fetching link: %w", err)
	}
	if link == nil {
		return domain.ErrLinkNotFound
	}
	if link.CreatedBy != nil && *link.CreatedBy != in.UserID {
		return domain.ErrLinkNotFound
	}

	if err := s.linkRepo.Delete(ctx, in.Slug); err != nil {
		return fmt.Errorf("deleting link: %w", err)
	}

	return nil
}

// ─── Admin ───────────────────────────────────────────────────────────────────

func (s *LinkService) ListLinks(ctx context.Context, in services.AdminListLinksInput) (services.AdminListLinksOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}

	var opts []repository.LinkOption
	opts = append(opts, repository.WithPagination(limit, in.Offset))

	links, err := s.linkRepo.List(ctx, opts...)
	if err != nil {
		return services.AdminListLinksOutput{}, fmt.Errorf("listing links: %w", err)
	}

	summaries := make([]services.AdminLinkSummary, len(links))
	for i, l := range links {
		summaries[i] = services.AdminLinkSummary{
			Slug:      l.Slug,
			Status:    string(l.Status),
			CreatedBy: l.CreatedBy,
			ExpiresAt: l.ExpiresAt,
			CreatedAt: l.CreatedAt,
		}
	}

	return services.AdminListLinksOutput{
		Links:  summaries,
		Total:  len(summaries),
		Limit:  limit,
		Offset: in.Offset,
	}, nil
}

func (s *LinkService) GetLink(ctx context.Context, in services.AdminGetLinkInput) (services.AdminGetLinkOutput, error) {
	link, err := s.linkRepo.GetBySlug(ctx, in.ID)
	if err != nil {
		return services.AdminGetLinkOutput{}, fmt.Errorf("fetching link: %w", err)
	}
	if link == nil {
		return services.AdminGetLinkOutput{}, domain.ErrLinkNotFound
	}

	return services.AdminGetLinkOutput{
		Link: services.AdminLinkDetail{
			ID:        link.ID,
			Slug:      link.Slug,
			Status:    string(link.Status),
			CreatedBy: link.CreatedBy,
			ExpiresAt: link.ExpiresAt,
			CreatedAt: link.CreatedAt,
		},
	}, nil
}
