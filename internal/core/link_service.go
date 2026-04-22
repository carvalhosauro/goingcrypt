package core

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
)

type LinkService struct {
	linkRepo   repository.LinkRepository
	transactor repository.Transactor
	generator  ports.Generator
}

func NewLinkService(linkRepo repository.LinkRepository, transactor repository.Transactor, generator ports.Generator) *LinkService {
	return &LinkService{linkRepo: linkRepo, transactor: transactor, generator: generator}
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

	accessLog := &domain.LinkAccessLog{
		LinkID:    link.ID,
		IPAddress: in.IPAddress,
		UserAgent: in.UserAgent,
		OpenedAt:  time.Now(),
	}

	if err := s.transactor.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.linkRepo.Update(txCtx, link); err != nil {
			return fmt.Errorf("updating link status: %w", err)
		}
		if err := s.linkRepo.CreateAccessLog(txCtx, accessLog); err != nil {
			return fmt.Errorf("creating access log: %w", err)
		}
		return nil
	}); err != nil {
		return services.AccessLinkOutput{}, err
	}

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
		ID: id,
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
