package authservice

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type quietSession struct {
	deps   *testDeps
	token  string
	stored *session.Session
}

func newQuietSession(t *testing.T, issuedBySignIn bool) *quietSession {
	t.Helper()

	deps := setupTest(t)
	sessionID := pulid.MustNew("ses_")
	issued, err := sessiontoken.Issue(sessionID)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.LastAccessedAt = time.Now().Add(-time.Hour).Unix()
	stored.IssuedBySignIn = issuedBySignIn
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)

	return &quietSession{deps: deps, token: issued.Token, stored: stored}
}

func (q *quietSession) expectMember(user *tenant.User, err error) {
	q.deps.userRepo.On("GetTenantMember", mock.Anything, mock.MatchedBy(
		func(req tenantports.GetTenantMemberRequest) bool {
			return req.UserID == q.stored.UserID &&
				req.TenantInfo.OrgID == q.stored.OrganizationID &&
				req.TenantInfo.BuID == q.stored.BusinessUnitID &&
				req.Now > 0
		},
	)).Return(user, err).Once()
}

func TestAuthenticateSession_KeepsASessionThatStillHasAccess(t *testing.T) {
	t.Parallel()

	q := newQuietSession(t, true)
	q.expectMember(&tenant.User{Status: domaintypes.StatusActive}, nil)
	q.deps.sessionRepo.On("Touch", mock.Anything, q.stored.ID, mock.AnythingOfType("int64")).
		Return(nil).Once()

	sess, err := q.deps.svc.AuthenticateSession(t.Context(), q.token)

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.Unix(sess.LastAccessedAt, 0), 5*time.Second)
}

func TestAuthenticateSession_EndsASessionWhoseMembershipEnded(t *testing.T) {
	t.Parallel()

	q := newQuietSession(t, true)
	q.expectMember(nil, errortypes.NewNotFoundError("User not found"))
	q.deps.sessionRepo.On("Delete", mock.Anything, q.stored.ID).Return(nil).Once()

	sess, err := q.deps.svc.AuthenticateSession(t.Context(), q.token)

	require.Error(t, err)
	assert.Nil(t, sess)
	var authErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authErr)
	q.deps.sessionRepo.AssertNotCalled(t, "Touch", mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthenticateSession_EndsASessionWhoseUserWasDeactivated(t *testing.T) {
	t.Parallel()

	q := newQuietSession(t, true)
	q.expectMember(&tenant.User{Status: domaintypes.StatusInactive}, nil)
	q.deps.sessionRepo.On("Delete", mock.Anything, q.stored.ID).Return(nil).Once()

	_, err := q.deps.svc.AuthenticateSession(t.Context(), q.token)

	require.Error(t, err)
}

func TestAuthenticateSession_EndsASessionWhoseUserWasLocked(t *testing.T) {
	t.Parallel()

	q := newQuietSession(t, true)
	q.expectMember(&tenant.User{Status: domaintypes.StatusActive, IsLocked: true}, nil)
	q.deps.sessionRepo.On("Delete", mock.Anything, q.stored.ID).Return(nil).Once()

	_, err := q.deps.svc.AuthenticateSession(t.Context(), q.token)

	require.Error(t, err)
}

func TestAuthenticateSession_RetriesWhenAccessCannotBeConfirmed(t *testing.T) {
	t.Parallel()

	q := newQuietSession(t, true)
	q.expectMember(nil, errors.New("database unavailable"))

	sess, err := q.deps.svc.AuthenticateSession(t.Context(), q.token)

	require.NoError(t, err)
	require.NotNil(t, sess)
	q.deps.sessionRepo.AssertNotCalled(t, "Touch", mock.Anything, mock.Anything, mock.Anything)
	q.deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestAuthenticateSession_LeavesSessionsItDidNotIssueToTheirIssuer(t *testing.T) {
	t.Parallel()

	q := newQuietSession(t, false)
	q.deps.sessionRepo.On("Touch", mock.Anything, q.stored.ID, mock.AnythingOfType("int64")).
		Return(nil).Once()

	_, err := q.deps.svc.AuthenticateSession(t.Context(), q.token)

	require.NoError(t, err)
	q.deps.userRepo.AssertNotCalled(t, "GetTenantMember", mock.Anything, mock.Anything)
}
