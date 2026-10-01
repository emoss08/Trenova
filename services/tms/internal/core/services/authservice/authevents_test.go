package authservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingAuthEvents struct {
	mu      sync.Mutex
	records []services.AuthEventRecord
}

func (r *recordingAuthEvents) Record(_ context.Context, rec *services.AuthEventRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, *rec)
}

func (r *recordingAuthEvents) only(t *testing.T) services.AuthEventRecord {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Len(t, r.records, 1)
	return r.records[0]
}

func TestLoginRecordsASuccessfulSignIn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "password123",
	})
	require.NoError(t, err)

	rec := deps.authEvents.only(t)
	assert.Equal(t, services.AuthEventProviderPassword, rec.Provider)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, rec.Outcome)
	assert.Equal(t, usr.ID, rec.UserID)
	assert.Equal(t, usr.CurrentOrganizationID, rec.OrganizationID)
	assert.Equal(t, usr.BusinessUnitID, rec.BusinessUnitID)
	assert.Empty(t, rec.ErrorCode)
}

func TestLoginRecordsAnUnknownAccountWithoutATenant(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	deps.userRepo.On("FindByEmail", mock.Anything, "nobody@example.com").
		Return(nil, errors.New("not found"))

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: "nobody@example.com",
		Password:     "password123",
	})
	require.Error(t, err)

	rec := deps.authEvents.only(t)
	assert.Equal(t, iam.AuthEventOutcomeFailed, rec.Outcome)
	assert.Equal(t, authErrorUnknownAccount, rec.ErrorCode)
	assert.True(t, rec.UserID.IsNil())
	assert.True(t, rec.OrganizationID.IsNil())
}

func TestLoginRecordsAWrongPasswordAgainstTheAccount(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)

	deps.userRepo.On("FindByEmail", mock.Anything, usr.EmailAddress).Return(usr, nil)

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: usr.EmailAddress,
		Password:     "wrong-password",
	})
	require.Error(t, err)

	rec := deps.authEvents.only(t)
	assert.Equal(t, iam.AuthEventOutcomeFailed, rec.Outcome)
	assert.Equal(t, authErrorRejectedLogin, rec.ErrorCode)
	assert.Equal(t, usr.ID, rec.UserID)
	assert.Equal(t, usr.CurrentOrganizationID, rec.OrganizationID)
}

func TestLoginDoesNotRecordARequestThatFailsValidation(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.Login(t.Context(), services.LoginRequest{})
	require.Error(t, err)

	assert.Empty(t, deps.authEvents.records)
}

func TestLogoutRecordsTheSessionOwner(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	sess := &session.Session{
		ID:             pulid.MustNew("ses_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	deps.sessionRepo.On("Get", mock.Anything, sess.ID).Return(sess, nil)
	deps.sessionRepo.On("Delete", mock.Anything, sess.ID).Return(nil)

	require.NoError(t, deps.svc.Logout(t.Context(), sess.ID))

	rec := deps.authEvents.only(t)
	assert.Equal(t, services.AuthEventProviderLogout, rec.Provider)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, rec.Outcome)
	assert.Equal(t, sess.UserID, rec.UserID)
	assert.Equal(t, sess.OrganizationID, rec.OrganizationID)
}

func TestAuthOutcomeClassifiesErrors(t *testing.T) {
	t.Parallel()

	assert.Equal(t, iam.AuthEventOutcomeSuccess, authOutcome(nil))
	assert.Equal(t, iam.AuthEventOutcomeDenied, authOutcome(errSSORequired))
	assert.Equal(t, iam.AuthEventOutcomeDenied, authOutcome(errortypes.NewAuthorizationError("no")))
	assert.Equal(t, iam.AuthEventOutcomeFailed, authOutcome(errInvalidCredentials))
	assert.Equal(t, authErrorSSORequired, authErrorCode(errSSORequired))
	assert.Equal(t, authErrorAuthentication, authErrorCode(errInvalidCredentials))
	assert.Equal(t, authErrorInternal, authErrorCode(errors.New("boom")))
}
