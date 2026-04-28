package core_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/core"
	"github.com/carvalhosauro/goingcrypt/internal/core/mocks"
	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func newService(repo *mocks.LinkRepository, gen *mocks.Generator) *core.LinkService {
	return core.NewLinkService(repo, gen)
}

// ─── AccessLink ──────────────────────────────────────────────────────────────

func TestAccessLink_HappyPath(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	key := "secret"
	logID := uuid.New()
	link := &domain.Link{
		ID:           uuid.New(),
		Slug:         "abc123",
		HashedKey:    hashKey(key),
		CipheredText: "encrypted-data",
		Status:       domain.StatusWaiting,
	}

	repo.On("GetBySlug", mock.Anything, "abc123").Return(link, nil)
	repo.On("Update", mock.Anything, link).Return(nil)
	// GenerateUUID and CreateAccessLog are called in background goroutine.
	gen.On("GenerateUUID", mock.Anything).Return(logID, nil).Maybe()
	repo.On("CreateAccessLog", mock.Anything, mock.AnythingOfType("*domain.LinkAccessLog")).Return(nil).Maybe()

	out, err := svc.AccessLink(context.Background(), services.AccessLinkInput{
		Slug: "abc123", Key: key,
	})

	assert.NoError(t, err)
	assert.Equal(t, "encrypted-data", out.CipheredText)
	// Give the background goroutine a moment to flush.
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestAccessLink_SlugNotFound(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	repo.On("GetBySlug", mock.Anything, "missing").Return(nil, nil)

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "missing", Key: "k"})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	repo.AssertExpectations(t)
}

func TestAccessLink_GetBySlugError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	dbErr := errors.New("connection refused")
	repo.On("GetBySlug", mock.Anything, "abc").Return(nil, dbErr)

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "abc", Key: "k"})

	assert.ErrorContains(t, err, "fetching link")
	assert.ErrorIs(t, err, dbErr)
}

func TestAccessLink_LinkAlreadyOpened(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	link := &domain.Link{
		Slug:      "abc",
		HashedKey: hashKey("k"),
		Status:    domain.StatusOpened,
	}
	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "abc", Key: "k"})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	// Update must NOT be called: Invalidate is only triggered when IsExpired()
	repo.AssertNotCalled(t, "Update")
}

func TestAccessLink_LinkWaitingButExpired_InvalidatesAndUpdates(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	past := time.Now().Add(-1 * time.Hour)
	link := &domain.Link{
		Slug:      "abc",
		HashedKey: hashKey("k"),
		Status:    domain.StatusWaiting,
		ExpiresAt: &past,
	}
	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)
	repo.On("Update", mock.Anything, link).Return(nil) // best-effort, called once

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "abc", Key: "k"})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	assert.Equal(t, domain.StatusExpired, link.Status)
	repo.AssertCalled(t, "Update", mock.Anything, link)
}

func TestAccessLink_LinkStatusExpiredNoExpiresAt_SkipsInvalidate(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	link := &domain.Link{
		Slug:      "abc",
		HashedKey: hashKey("k"),
		Status:    domain.StatusExpired,
		// ExpiresAt nil → IsExpired() = false
	}
	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "abc", Key: "k"})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	repo.AssertNotCalled(t, "Update")
}

func TestAccessLink_WrongKey(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	link := &domain.Link{
		Slug:      "abc",
		HashedKey: hashKey("correct-key"),
		Status:    domain.StatusWaiting,
	}
	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "abc", Key: "wrong-key"})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	repo.AssertNotCalled(t, "Update")
}

func TestAccessLink_UpdateError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	key := "secret"
	link := &domain.Link{
		Slug:         "abc",
		HashedKey:    hashKey(key),
		CipheredText: "data",
		Status:       domain.StatusWaiting,
	}
	dbErr := errors.New("update failed")

	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)
	repo.On("Update", mock.Anything, link).Return(dbErr)

	_, err := svc.AccessLink(context.Background(), services.AccessLinkInput{Slug: "abc", Key: key})

	assert.ErrorContains(t, err, "updating link status")
	assert.ErrorIs(t, err, dbErr)
}

// ─── CreateLink ──────────────────────────────────────────────────────────────

func TestCreateLink_HappyPath(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	id := uuid.New()
	gen.On("GenerateUUID", mock.Anything).Return(id, nil)
	gen.On("GenerateSlug", mock.Anything, id.String()).Return("slug-xyz", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(l *domain.Link) bool {
		return l.Slug == "slug-xyz" &&
			l.HashedKey == hashKey("mykey") &&
			l.CipheredText == "cipher" &&
			l.ExpiresAt == nil
	})).Return(nil)

	out, err := svc.CreateLink(context.Background(), services.CreateLinkInput{
		Key: "mykey", CipheredText: "cipher",
	})

	assert.NoError(t, err)
	assert.Equal(t, "slug-xyz", out.Slug)
	gen.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestCreateLink_WithExpiresIn(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	id := uuid.New()
	duration := 24 * time.Hour
	before := time.Now()

	gen.On("GenerateUUID", mock.Anything).Return(id, nil)
	gen.On("GenerateSlug", mock.Anything, id.String()).Return("slug-exp", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(l *domain.Link) bool {
		if l.ExpiresAt == nil {
			return false
		}
		after := time.Now()
		return l.ExpiresAt.After(before.Add(duration-time.Second)) &&
			l.ExpiresAt.Before(after.Add(duration+time.Second))
	})).Return(nil)

	out, err := svc.CreateLink(context.Background(), services.CreateLinkInput{
		Key: "k", CipheredText: "c", ExpiresIn: &duration,
	})

	assert.NoError(t, err)
	assert.Equal(t, "slug-exp", out.Slug)
}

func TestCreateLink_KeyIsHashed(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	id := uuid.New()
	gen.On("GenerateUUID", mock.Anything).Return(id, nil)
	gen.On("GenerateSlug", mock.Anything, id.String()).Return("s", nil)

	var captured *domain.Link
	repo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(1).(*domain.Link)
	}).Return(nil)

	_, _ = svc.CreateLink(context.Background(), services.CreateLinkInput{Key: "plaintext", CipheredText: "c"})

	assert.Equal(t, hashKey("plaintext"), captured.HashedKey)
	assert.NotEqual(t, "plaintext", captured.HashedKey)
}

func TestCreateLink_GenerateUUIDError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	genErr := errors.New("uuid gen failed")
	gen.On("GenerateUUID", mock.Anything).Return(uuid.UUID{}, genErr)

	_, err := svc.CreateLink(context.Background(), services.CreateLinkInput{Key: "k", CipheredText: "c"})

	assert.ErrorContains(t, err, "generating uuid")
	assert.ErrorIs(t, err, genErr)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateLink_GenerateSlugError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	id := uuid.New()
	slugErr := errors.New("slug gen failed")
	gen.On("GenerateUUID", mock.Anything).Return(id, nil)
	gen.On("GenerateSlug", mock.Anything, id.String()).Return("", slugErr)

	_, err := svc.CreateLink(context.Background(), services.CreateLinkInput{Key: "k", CipheredText: "c"})

	assert.ErrorContains(t, err, "generating slug")
	assert.ErrorIs(t, err, slugErr)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateLink_RepositoryCreateError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	id := uuid.New()
	createErr := errors.New("db error")
	gen.On("GenerateUUID", mock.Anything).Return(id, nil)
	gen.On("GenerateSlug", mock.Anything, id.String()).Return("slug", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(createErr)

	_, err := svc.CreateLink(context.Background(), services.CreateLinkInput{Key: "k", CipheredText: "c"})

	assert.ErrorContains(t, err, "creating link")
	assert.ErrorIs(t, err, createErr)
}

// ─── DeleteLink ──────────────────────────────────────────────────────────────

func TestDeleteLink_HappyPath(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	userID := uuid.New()
	link := &domain.Link{Slug: "abc", CreatedBy: &userID}

	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)
	repo.On("Delete", mock.Anything, "abc").Return(nil)

	err := svc.DeleteLink(context.Background(), services.DeleteLinkInput{Slug: "abc", UserID: userID})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteLink_LinkNotFound(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	repo.On("GetBySlug", mock.Anything, "missing").Return(nil, nil)

	err := svc.DeleteLink(context.Background(), services.DeleteLinkInput{Slug: "missing", UserID: uuid.New()})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	repo.AssertNotCalled(t, "Delete")
}

func TestDeleteLink_GetBySlugError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	dbErr := errors.New("db down")
	repo.On("GetBySlug", mock.Anything, "abc").Return(nil, dbErr)

	err := svc.DeleteLink(context.Background(), services.DeleteLinkInput{Slug: "abc", UserID: uuid.New()})

	assert.ErrorContains(t, err, "fetching link")
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteLink_DifferentOwner(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	owner := uuid.New()
	caller := uuid.New()
	link := &domain.Link{Slug: "abc", CreatedBy: &owner}

	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)

	err := svc.DeleteLink(context.Background(), services.DeleteLinkInput{Slug: "abc", UserID: caller})

	assert.ErrorIs(t, err, domain.ErrLinkNotFound)
	repo.AssertNotCalled(t, "Delete")
}

func TestDeleteLink_NilCreatedBy_AnyUserCanDelete(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	link := &domain.Link{Slug: "abc", CreatedBy: nil}

	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)
	repo.On("Delete", mock.Anything, "abc").Return(nil)

	err := svc.DeleteLink(context.Background(), services.DeleteLinkInput{Slug: "abc", UserID: uuid.New()})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteLink_RepositoryDeleteError(t *testing.T) {
	repo := &mocks.LinkRepository{}
	gen := &mocks.Generator{}
	svc := newService(repo, gen)

	userID := uuid.New()
	link := &domain.Link{Slug: "abc", CreatedBy: &userID}
	delErr := errors.New("delete failed")

	repo.On("GetBySlug", mock.Anything, "abc").Return(link, nil)
	repo.On("Delete", mock.Anything, "abc").Return(delErr)

	err := svc.DeleteLink(context.Background(), services.DeleteLinkInput{Slug: "abc", UserID: userID})

	assert.ErrorContains(t, err, "deleting link")
	assert.ErrorIs(t, err, delErr)
}
