package core_test

import (
	"context"
	"database/sql"
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

type authFixture struct {
	userRepo  *mocks.UserRepository
	tokenRepo *mocks.RefreshTokenRepository
	tx        *mocks.Transactor
	gen       *mocks.Generator
	hasher    *mocks.Hasher
	tokenMgr  *mocks.TokenManager
	totp      *mocks.TOTPManager
	svc       *core.AuthService
}

func newAuthFixture() *authFixture {
	f := &authFixture{
		userRepo:  &mocks.UserRepository{},
		tokenRepo: &mocks.RefreshTokenRepository{},
		tx:        &mocks.Transactor{},
		gen:       &mocks.Generator{},
		hasher:    &mocks.Hasher{},
		tokenMgr:  &mocks.TokenManager{},
		totp:      &mocks.TOTPManager{},
	}
	f.svc = core.NewAuthService(
		f.userRepo, f.tokenRepo, f.tx, f.gen, f.hasher, f.tokenMgr, f.totp, "goingcrypt",
	)
	return f
}

// setupTokenPair sets up the mocks for issueTokenPair (GenerateAccessToken + storeRefreshToken).
func (f *authFixture) setupTokenPair() {
	f.tokenMgr.On("GenerateAccessToken", mock.Anything, mock.Anything).Return("access-token", nil).Once()
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil).Once()
}

func assertExpectations(t *testing.T, f *authFixture) {
	t.Helper()
	f.userRepo.AssertExpectations(t)
	f.tokenRepo.AssertExpectations(t)
	f.tx.AssertExpectations(t)
	f.gen.AssertExpectations(t)
	f.hasher.AssertExpectations(t)
	f.tokenMgr.AssertExpectations(t)
	f.totp.AssertExpectations(t)
}

// ─── SignUp ───────────────────────────────────────────────────────────────────

func TestSignUp_HappyPath(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()

	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)
	f.gen.On("GenerateUUID", mock.Anything).Return(userID, nil).Once()
	f.hasher.On("Hash", mock.Anything, "password123").Return("hashed-pw", nil)
	f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	f.setupTokenPair()

	out, err := f.svc.SignUp(context.Background(), services.SignUpInput{
		Username: "alice", Password: "password123",
	})

	assert.NoError(t, err)
	assert.Equal(t, userID, out.UserID)
	assert.Equal(t, "access-token", out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assertExpectations(t, f)
}

func TestSignUp_GetByUsernameError(t *testing.T) {
	f := newAuthFixture()
	dbErr := errors.New("db down")
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, dbErr)

	_, err := f.svc.SignUp(context.Background(), services.SignUpInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "checking username")
	assert.ErrorIs(t, err, dbErr)
}

func TestSignUp_UsernameTaken(t *testing.T) {
	f := newAuthFixture()
	existing := &domain.User{ID: uuid.New(), Username: "alice"}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(existing, nil)

	_, err := f.svc.SignUp(context.Background(), services.SignUpInput{Username: "alice", Password: "pw"})

	assert.ErrorIs(t, err, domain.ErrUsernameTaken)
	f.gen.AssertNotCalled(t, "GenerateUUID")
}

func TestSignUp_GenerateUUIDError(t *testing.T) {
	f := newAuthFixture()
	genErr := errors.New("uuid failed")
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.UUID{}, genErr).Once()

	_, err := f.svc.SignUp(context.Background(), services.SignUpInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "generating uuid")
	assert.ErrorIs(t, err, genErr)
}

func TestSignUp_HashError(t *testing.T) {
	f := newAuthFixture()
	hashErr := errors.New("argon2 failed")
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.hasher.On("Hash", mock.Anything, "pw").Return("", hashErr)

	_, err := f.svc.SignUp(context.Background(), services.SignUpInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "hashing password")
	assert.ErrorIs(t, err, hashErr)
}

func TestSignUp_CreateUserError(t *testing.T) {
	f := newAuthFixture()
	createErr := errors.New("constraint violation")
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.hasher.On("Hash", mock.Anything, "pw").Return("hashed", nil)
	f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(createErr)

	_, err := f.svc.SignUp(context.Background(), services.SignUpInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "creating user")
	assert.ErrorIs(t, err, createErr)
}

func TestSignUp_GenerateAccessTokenError(t *testing.T) {
	f := newAuthFixture()
	tokenErr := errors.New("jwt signing failed")
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.hasher.On("Hash", mock.Anything, "pw").Return("hashed", nil)
	f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	f.tokenMgr.On("GenerateAccessToken", mock.Anything, mock.Anything).Return("", tokenErr).Once()

	_, err := f.svc.SignUp(context.Background(), services.SignUpInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "generating access token")
	assert.ErrorIs(t, err, tokenErr)
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLogin_HappyPath(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{ID: uuid.New(), Username: "alice", Password: "hashed-pw"}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "password123", "hashed-pw").Return(true, nil)
	f.setupTokenPair()

	out, err := f.svc.Login(context.Background(), services.LoginInput{
		Username: "alice", Password: "password123",
	})

	assert.NoError(t, err)
	assert.Equal(t, "access-token", out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assert.False(t, out.MFARequired)
	assertExpectations(t, f)
}

func TestLogin_GetByUsernameError(t *testing.T) {
	f := newAuthFixture()
	dbErr := errors.New("db down")
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, dbErr)

	_, err := f.svc.Login(context.Background(), services.LoginInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "fetching user")
	assert.ErrorIs(t, err, dbErr)
}

func TestLogin_UserNotFound(t *testing.T) {
	f := newAuthFixture()
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)

	_, err := f.svc.Login(context.Background(), services.LoginInput{Username: "alice", Password: "pw"})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_UserDeleted(t *testing.T) {
	f := newAuthFixture()
	deleted := &domain.User{ID: uuid.New(), Username: "alice"}
	deleted.Delete()
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(deleted, nil)

	_, err := f.svc.Login(context.Background(), services.LoginInput{Username: "alice", Password: "pw"})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_VerifyPasswordError(t *testing.T) {
	f := newAuthFixture()
	verifyErr := errors.New("argon2 error")
	user := &domain.User{ID: uuid.New(), Password: "hashed-pw"}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "pw", "hashed-pw").Return(false, verifyErr)

	_, err := f.svc.Login(context.Background(), services.LoginInput{Username: "alice", Password: "pw"})

	assert.ErrorContains(t, err, "verifying password")
	assert.ErrorIs(t, err, verifyErr)
}

func TestLogin_WrongPassword(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{ID: uuid.New(), Password: "hashed-pw"}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "wrong", "hashed-pw").Return(false, nil)

	_, err := f.svc.Login(context.Background(), services.LoginInput{Username: "alice", Password: "wrong"})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_MFARequired(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, Password: "hashed-pw", MfaEnabled: true}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "pw", "hashed-pw").Return(true, nil)

	out, err := f.svc.Login(context.Background(), services.LoginInput{Username: "alice", Password: "pw"})

	assert.NoError(t, err)
	assert.True(t, out.MFARequired)
	assert.Equal(t, userID, out.UserID)
	assert.Empty(t, out.AccessToken)
	f.tokenMgr.AssertNotCalled(t, "GenerateAccessToken")
}

// ─── LoginWithMFA ─────────────────────────────────────────────────────────────

func TestLoginWithMFA_HappyPath(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{
		ID:         uuid.New(),
		Password:   "hashed-pw",
		MfaEnabled: true,
		MfaSecret:  sql.NullString{String: "TOTP_SECRET", Valid: true},
	}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "pw", "hashed-pw").Return(true, nil)
	f.totp.On("Validate", mock.Anything, "TOTP_SECRET", "123456").Return(true)
	f.setupTokenPair()

	out, err := f.svc.LoginWithMFA(context.Background(), services.LoginWithMFAInput{
		Username: "alice", Password: "pw", Code: "123456",
	})

	assert.NoError(t, err)
	assert.Equal(t, "access-token", out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assertExpectations(t, f)
}

func TestLoginWithMFA_UserNotFound(t *testing.T) {
	f := newAuthFixture()
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(nil, nil)

	_, err := f.svc.LoginWithMFA(context.Background(), services.LoginWithMFAInput{
		Username: "alice", Password: "pw", Code: "000000",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLoginWithMFA_WrongPassword(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{ID: uuid.New(), Password: "hashed-pw"}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "wrong", "hashed-pw").Return(false, nil)

	_, err := f.svc.LoginWithMFA(context.Background(), services.LoginWithMFAInput{
		Username: "alice", Password: "wrong", Code: "123456",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLoginWithMFA_MFANotEnabled(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{ID: uuid.New(), Password: "hashed-pw", MfaEnabled: false}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "pw", "hashed-pw").Return(true, nil)

	_, err := f.svc.LoginWithMFA(context.Background(), services.LoginWithMFAInput{
		Username: "alice", Password: "pw", Code: "123456",
	})

	assert.ErrorIs(t, err, domain.ErrMFANotEnabled)
}

func TestLoginWithMFA_SecretNotValid(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{
		ID:         uuid.New(),
		Password:   "hashed-pw",
		MfaEnabled: true,
		MfaSecret:  sql.NullString{Valid: false},
	}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "pw", "hashed-pw").Return(true, nil)

	_, err := f.svc.LoginWithMFA(context.Background(), services.LoginWithMFAInput{
		Username: "alice", Password: "pw", Code: "123456",
	})

	assert.ErrorIs(t, err, domain.ErrMFANotEnabled)
}

func TestLoginWithMFA_InvalidCode(t *testing.T) {
	f := newAuthFixture()
	user := &domain.User{
		ID:         uuid.New(),
		Password:   "hashed-pw",
		MfaEnabled: true,
		MfaSecret:  sql.NullString{String: "SECRET", Valid: true},
	}
	f.userRepo.On("GetByUsername", mock.Anything, "alice").Return(user, nil)
	f.hasher.On("Verify", mock.Anything, "pw", "hashed-pw").Return(true, nil)
	f.totp.On("Validate", mock.Anything, "SECRET", "000000").Return(false)

	_, err := f.svc.LoginWithMFA(context.Background(), services.LoginWithMFAInput{
		Username: "alice", Password: "pw", Code: "000000",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidMFACode)
}

// ─── RefreshTokens ────────────────────────────────────────────────────────────

func validRefreshToken(userID uuid.UUID) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hashKey("raw-token"),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
}

func TestRefreshTokens_HappyPath(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	stored := validRefreshToken(userID)

	f.tokenRepo.On("GetByHash", mock.Anything, hashKey("raw-token")).Return(stored, nil)
	f.tx.On("RunInTx", mock.Anything, mock.Anything).Return(nil)
	// GenerateUUID: once for newID (ReplacedBy), once inside storeRefreshToken
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.tokenMgr.On("GenerateAccessToken", mock.Anything, mock.Anything).Return("access-token", nil).Once()
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil).Once()
	f.tokenRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil).Once()

	out, err := f.svc.RefreshTokens(context.Background(), services.RefreshTokensInput{
		RefreshToken: "raw-token",
	})

	assert.NoError(t, err)
	assert.Equal(t, "access-token", out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assertExpectations(t, f)
}

func TestRefreshTokens_GetByHashError(t *testing.T) {
	f := newAuthFixture()
	dbErr := errors.New("db down")
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(nil, dbErr)

	_, err := f.svc.RefreshTokens(context.Background(), services.RefreshTokensInput{RefreshToken: "tok"})

	assert.ErrorContains(t, err, "fetching refresh token")
	assert.ErrorIs(t, err, dbErr)
}

func TestRefreshTokens_TokenNotFound(t *testing.T) {
	f := newAuthFixture()
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(nil, nil)

	_, err := f.svc.RefreshTokens(context.Background(), services.RefreshTokensInput{RefreshToken: "tok"})

	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestRefreshTokens_TokenRevoked(t *testing.T) {
	f := newAuthFixture()
	stored := validRefreshToken(uuid.New())
	stored.Revoke()
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(stored, nil)

	_, err := f.svc.RefreshTokens(context.Background(), services.RefreshTokensInput{RefreshToken: "tok"})

	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestRefreshTokens_TokenExpired(t *testing.T) {
	f := newAuthFixture()
	stored := validRefreshToken(uuid.New())
	stored.ExpiresAt = time.Now().Add(-time.Hour)
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(stored, nil)

	_, err := f.svc.RefreshTokens(context.Background(), services.RefreshTokensInput{RefreshToken: "tok"})

	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestRefreshTokens_UpdateOldTokenError(t *testing.T) {
	f := newAuthFixture()
	stored := validRefreshToken(uuid.New())
	updateErr := errors.New("update failed")

	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(stored, nil)
	f.tx.On("RunInTx", mock.Anything, mock.Anything).Return(nil)
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.tokenMgr.On("GenerateAccessToken", mock.Anything, mock.Anything).Return("access-token", nil).Once()
	f.gen.On("GenerateUUID", mock.Anything).Return(uuid.New(), nil).Once()
	f.tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil).Once()
	f.tokenRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(updateErr).Once()

	_, err := f.svc.RefreshTokens(context.Background(), services.RefreshTokensInput{RefreshToken: "tok"})

	assert.ErrorContains(t, err, "revoking old refresh token")
	assert.ErrorIs(t, err, updateErr)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogout_HappyPath(t *testing.T) {
	f := newAuthFixture()
	stored := validRefreshToken(uuid.New())
	f.tokenRepo.On("GetByHash", mock.Anything, hashKey("raw-token")).Return(stored, nil)
	f.tokenRepo.On("Update", mock.Anything, stored).Return(nil)

	err := f.svc.Logout(context.Background(), services.LogoutInput{RefreshToken: "raw-token"})

	assert.NoError(t, err)
	assert.NotNil(t, stored.RevokedAt)
	assertExpectations(t, f)
}

func TestLogout_GetByHashError(t *testing.T) {
	f := newAuthFixture()
	dbErr := errors.New("db down")
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(nil, dbErr)

	err := f.svc.Logout(context.Background(), services.LogoutInput{RefreshToken: "tok"})

	assert.ErrorContains(t, err, "fetching refresh token")
	assert.ErrorIs(t, err, dbErr)
}

func TestLogout_TokenNotFound(t *testing.T) {
	f := newAuthFixture()
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(nil, nil)

	err := f.svc.Logout(context.Background(), services.LogoutInput{RefreshToken: "tok"})

	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestLogout_TokenAlreadyRevoked(t *testing.T) {
	f := newAuthFixture()
	stored := validRefreshToken(uuid.New())
	stored.Revoke()
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(stored, nil)

	err := f.svc.Logout(context.Background(), services.LogoutInput{RefreshToken: "tok"})

	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	f.tokenRepo.AssertNotCalled(t, "Update")
}

func TestLogout_UpdateError(t *testing.T) {
	f := newAuthFixture()
	stored := validRefreshToken(uuid.New())
	updateErr := errors.New("update failed")
	f.tokenRepo.On("GetByHash", mock.Anything, mock.Anything).Return(stored, nil)
	f.tokenRepo.On("Update", mock.Anything, stored).Return(updateErr)

	err := f.svc.Logout(context.Background(), services.LogoutInput{RefreshToken: "raw"})

	assert.ErrorContains(t, err, "revoking token")
	assert.ErrorIs(t, err, updateErr)
}

// ─── EnableMFA ────────────────────────────────────────────────────────────────

func TestEnableMFA_HappyPath(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, Username: "alice", MfaEnabled: false}
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	f.totp.On("GenerateSecret", mock.Anything).Return("BASE32SECRET", nil)
	f.totp.On("GenerateProvisioningURI", mock.Anything, "BASE32SECRET", "alice", "goingcrypt").
		Return("otpauth://totp/goingcrypt:alice?secret=BASE32SECRET")

	out, err := f.svc.EnableMFA(context.Background(), services.EnableMFAInput{UserID: userID})

	assert.NoError(t, err)
	assert.Equal(t, "BASE32SECRET", out.Secret)
	assert.Contains(t, out.ProvisioningURI, "BASE32SECRET")
	assertExpectations(t, f)
}

func TestEnableMFA_GetByIDError(t *testing.T) {
	f := newAuthFixture()
	dbErr := errors.New("db down")
	userID := uuid.New()
	f.userRepo.On("GetByID", mock.Anything, userID).Return(nil, dbErr)

	_, err := f.svc.EnableMFA(context.Background(), services.EnableMFAInput{UserID: userID})

	assert.ErrorContains(t, err, "fetching user")
	assert.ErrorIs(t, err, dbErr)
}

func TestEnableMFA_UserNotFound(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	f.userRepo.On("GetByID", mock.Anything, userID).Return(nil, nil)

	_, err := f.svc.EnableMFA(context.Background(), services.EnableMFAInput{UserID: userID})

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestEnableMFA_AlreadyEnabled(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, MfaEnabled: true}
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	_, err := f.svc.EnableMFA(context.Background(), services.EnableMFAInput{UserID: userID})

	assert.ErrorIs(t, err, domain.ErrMFAAlreadyEnabled)
	f.totp.AssertNotCalled(t, "GenerateSecret")
}

func TestEnableMFA_GenerateSecretError(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, MfaEnabled: false}
	secretErr := errors.New("entropy error")
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	f.totp.On("GenerateSecret", mock.Anything).Return("", secretErr)

	_, err := f.svc.EnableMFA(context.Background(), services.EnableMFAInput{UserID: userID})

	assert.ErrorContains(t, err, "generating TOTP secret")
	assert.ErrorIs(t, err, secretErr)
}

// ─── ConfirmMFA ───────────────────────────────────────────────────────────────

func TestConfirmMFA_HappyPath(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, MfaEnabled: false}
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	f.totp.On("Validate", mock.Anything, "SECRET", "123456").Return(true)
	f.userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.MfaEnabled && u.MfaSecret.Valid && u.MfaSecret.String == "SECRET"
	})).Return(nil)

	err := f.svc.ConfirmMFA(context.Background(), services.ConfirmMFAInput{
		UserID: userID, Secret: "SECRET", Code: "123456",
	})

	assert.NoError(t, err)
	assert.True(t, user.MfaEnabled)
	assertExpectations(t, f)
}

func TestConfirmMFA_GetByIDError(t *testing.T) {
	f := newAuthFixture()
	dbErr := errors.New("db down")
	userID := uuid.New()
	f.userRepo.On("GetByID", mock.Anything, userID).Return(nil, dbErr)

	err := f.svc.ConfirmMFA(context.Background(), services.ConfirmMFAInput{UserID: userID})

	assert.ErrorContains(t, err, "fetching user")
	assert.ErrorIs(t, err, dbErr)
}

func TestConfirmMFA_UserNotFound(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	f.userRepo.On("GetByID", mock.Anything, userID).Return(nil, nil)

	err := f.svc.ConfirmMFA(context.Background(), services.ConfirmMFAInput{UserID: userID})

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestConfirmMFA_AlreadyEnabled(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, MfaEnabled: true}
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	err := f.svc.ConfirmMFA(context.Background(), services.ConfirmMFAInput{UserID: userID, Secret: "S", Code: "C"})

	assert.ErrorIs(t, err, domain.ErrMFAAlreadyEnabled)
	f.totp.AssertNotCalled(t, "Validate")
}

func TestConfirmMFA_InvalidCode(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, MfaEnabled: false}
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	f.totp.On("Validate", mock.Anything, "SECRET", "000000").Return(false)

	err := f.svc.ConfirmMFA(context.Background(), services.ConfirmMFAInput{
		UserID: userID, Secret: "SECRET", Code: "000000",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidMFACode)
	f.userRepo.AssertNotCalled(t, "Update")
}

func TestConfirmMFA_UpdateError(t *testing.T) {
	f := newAuthFixture()
	userID := uuid.New()
	user := &domain.User{ID: userID, MfaEnabled: false}
	updateErr := errors.New("db error")
	f.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	f.totp.On("Validate", mock.Anything, "SECRET", "123456").Return(true)
	f.userRepo.On("Update", mock.Anything, mock.Anything).Return(updateErr)

	err := f.svc.ConfirmMFA(context.Background(), services.ConfirmMFAInput{
		UserID: userID, Secret: "SECRET", Code: "123456",
	})

	assert.ErrorContains(t, err, "enabling MFA")
	assert.ErrorIs(t, err, updateErr)
}
