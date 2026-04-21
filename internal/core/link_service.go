package core

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports"
	"github.com/carvalhosauro/goingcrypt/internal/ports/repository"
	"github.com/google/uuid"
)

var ErrLinkNotFound = errors.New("link not found or expired")

type LinkService struct {
	linkRepo   repository.LinkRepository
	transactor repository.Transactor
	generator ports.Generator
}

func NewLinkService(linkRepo repository.LinkRepository, transactor repository.Transactor) *LinkService {
	return &LinkService{linkRepo: linkRepo, transactor: transactor}
}

type AccessLinkInput struct {
	Slug      string
	Key       string
	IPAddress string
	UserAgent string
}

type AccessLinkOutput struct {
	CipheredText string
}

func (s *LinkService) AccessLink(ctx context.Context, in AccessLinkInput) (AccessLinkOutput, error) {
	link, err := s.linkRepo.GetBySlug(ctx, in.Slug)
	if err != nil {
		return AccessLinkOutput{}, fmt.Errorf("fetching link: %w", err)
	}

	if link == nil || !link.CanAccess() {
		if link != nil && link.IsExpired() {
			if err := link.Invalidate(); err == nil {
				_ = s.linkRepo.Update(ctx, link)
			}
		}
		return AccessLinkOutput{}, ErrLinkNotFound
	}

	hashedKey := sha256.Sum256([]byte(in.Key))
	hashedKeyHex := hex.EncodeToString(hashedKey[:])
	if subtle.ConstantTimeCompare([]byte(link.HashedKey), []byte(hashedKeyHex)) != 1 {
		return AccessLinkOutput{}, ErrLinkNotFound
	}

	cipheredText, err := link.Open()
	if err != nil {
		return AccessLinkOutput{}, fmt.Errorf("opening link: %w", err)
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
		return AccessLinkOutput{}, err
	}

	return AccessLinkOutput{CipheredText: cipheredText}, nil
}

type CreateLinkInput struct {
	Key          string
	CipheredText string
	ExpiresIn    *time.Duration
	CreatedBy    *uuid.UUID
}

type CreateLinkOutput struct {
	Slug string
}

func (s *LinkService) CreateLink(ctx context.Context, in CreateLinkInput) (CreateLinkOutput, error) {
	id, err := s.generator.GenerateUUID(ctx)
	if err != nil {
		return CreateLinkOutput{}, fmt.Errorf("generating uuid: %w", err)
	}

	slug, err := s.generator.GenerateSlug(ctx, id.String())
	if err != nil {
		return CreateLinkOutput{}, fmt.Errorf("generating slug: %w", err)
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
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return CreateLinkOutput{}, fmt.Errorf("creating link: %w", err)
	}

	return CreateLinkOutput{
		Slug: link.Slug,
	}, nil
}
