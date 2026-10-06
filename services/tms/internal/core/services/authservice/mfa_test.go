package authservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeMFA struct {
	services.MFAService
	active    bool
	validCode string
}

func (f *fakeMFA) HasActiveFactor(context.Context, pulid.ID) (bool, error) {
	return f.active, nil
}

func (f *fakeMFA) VerifySecondFactor(
	_ context.Context,
	req *services.VerifySecondFactorRequest,
) (string, error) {
	if req.Code == f.validCode {
		return services.MFAMethodTOTP, nil
	}
	return "", errortypes.NewValidationError("code", errortypes.ErrInvalid, "bad code")
}

type memoryChallenges struct {
	mu         sync.Mutex
	challenges map[string]*repositories.MFAChallenge
	failures   map[string]int64
}

func newMemoryChallenges() *memoryChallenges {
	return &memoryChallenges{
		challenges: map[string]*repositories.MFAChallenge{},
		failures:   map[string]int64{},
	}
}

func (m *memoryChallenges) Save(
	_ context.Context,
	hash string,
	challenge *repositories.MFAChallenge,
	_ time.Duration,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.challenges[hash] = challenge
	return nil
}

func (m *memoryChallenges) Get(
	_ context.Context,
	hash string,
) (*repositories.MFAChallenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	challenge, ok := m.challenges[hash]
	if !ok {
		return nil, errortypes.NewNotFoundError("missing")
	}
	return challenge, nil
}

func (m *memoryChallenges) Delete(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.challenges, hash)
	delete(m.failures, hash)
	return nil
}

func (m *memoryChallenges) RecordFailure(
	_ context.Context,
	hash string,
	_ time.Duration,
) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failures[hash]++
	return m.failures[hash], nil
}

func (m *memoryChallenges) has(hash string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.challenges[hash]
	return ok
}

func TestLoginWithASecondFactorIssuesAChallengeInsteadOfASession(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	challenges := newMemoryChallenges()
	deps.svc.mfa = &fakeMFA{active: true, validCode: "123456"}
	deps.svc.challenges = challenges

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)

	resp, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})

	require.NoError(t, err)
	assert.True(t, resp.MFARequired)
	assert.NotEmpty(t, resp.MFAChallengeToken)
	assert.Nil(t, resp.User)
	assert.Empty(t, resp.SessionID)
	assert.Empty(t, resp.SessionToken)
	assert.True(t, challenges.has(tokenutils.Hash(resp.MFAChallengeToken)))
	deps.sessionRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	rec := deps.authEvents.only(t)
	assert.Equal(t, mfaStateChallenged, rec.MFAState)
}

func TestVerifyMFAChallengeStartsAnAAL2Session(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	challenges := newMemoryChallenges()
	deps.svc.mfa = &fakeMFA{active: true, validCode: "123456"}
	deps.svc.challenges = challenges

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	login, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)

	var created *session.Session
	deps.userRepo.On("FindByIDForLogin", mock.Anything, usr.ID).Return(usr, nil)
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(1).(*session.Session) }).
		Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	resp, err := deps.svc.VerifyMFAChallenge(t.Context(), services.VerifyMFAChallengeRequest{
		ChallengeToken: login.MFAChallengeToken,
		Code:           "123456",
	})

	require.NoError(t, err)
	assert.False(t, resp.MFARequired)
	assert.Equal(t, usr, resp.User)
	require.NotNil(t, created)
	assert.Equal(t, 2, created.AuthenticatorAAL)
	assert.Positive(t, created.MFAAuthenticatedAt)
	assert.False(t, challenges.has(tokenutils.Hash(login.MFAChallengeToken)))

	_, err = deps.svc.VerifyMFAChallenge(t.Context(), services.VerifyMFAChallengeRequest{
		ChallengeToken: login.MFAChallengeToken,
		Code:           "123456",
	})
	require.ErrorIs(t, err, errMFAChallengeExpired)
}

func TestVerifyMFAChallengeGivesUpAfterRepeatedWrongCodes(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	challenges := newMemoryChallenges()
	deps.svc.mfa = &fakeMFA{active: true, validCode: "123456"}
	deps.svc.challenges = challenges

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	login, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)
	deps.userRepo.On("FindByIDForLogin", mock.Anything, usr.ID).Return(usr, nil)

	for range maxMFAChallengeFailures - 1 {
		_, err = deps.svc.VerifyMFAChallenge(t.Context(), services.VerifyMFAChallengeRequest{
			ChallengeToken: login.MFAChallengeToken,
			Code:           "000000",
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsError(err) || errortypes.IsMultiError(err))
	}

	_, err = deps.svc.VerifyMFAChallenge(t.Context(), services.VerifyMFAChallengeRequest{
		ChallengeToken: login.MFAChallengeToken,
		Code:           "000000",
	})
	require.ErrorIs(t, err, errMFAChallengeExhausted)
	assert.False(t, challenges.has(tokenutils.Hash(login.MFAChallengeToken)))
	deps.sessionRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestLoginWithoutASecondFactorStillSignsIn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	deps.svc.mfa = &fakeMFA{active: false}
	deps.svc.challenges = newMemoryChallenges()

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	resp, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})

	require.NoError(t, err)
	assert.False(t, resp.MFARequired)
	assert.NotEmpty(t, resp.SessionID)
}

func TestVerifyMFAChallengeRequiresAToken(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.VerifyMFAChallenge(t.Context(), services.VerifyMFAChallengeRequest{})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err) || errortypes.IsMultiError(err))
}
