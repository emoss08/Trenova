package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// turnStopper remembers whose replies it was asked to stop.
type turnStopper struct {
	asked []services.StopUserTurnsRequest
	err   error
}

func (s *turnStopper) StopAllForUser(_ context.Context, req services.StopUserTurnsRequest) error {
	s.asked = append(s.asked, req)

	return s.err
}

func signedIn() *session.Session {
	return &session.Session{
		ID:             pulid.MustNew("ses_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

// Once someone has signed out nobody is left to read their replies, and each
// one keeps running, and billing, until it is stopped.
func TestLogout_StopsThePersonsRepliesInProgress(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	stopper := &turnStopper{}
	deps.svc.turns = stopper
	sess := signedIn()

	deps.sessionRepo.On("Get", mock.Anything, sess.ID).Return(sess, nil)
	deps.sessionRepo.On("Delete", mock.Anything, sess.ID).Return(nil)

	require.NoError(t, deps.svc.Logout(t.Context(), sess.ID))

	require.Len(t, stopper.asked, 1)
	assert.Equal(t, sess.UserID, stopper.asked[0].UserID)
	assert.Equal(t, sess.OrganizationID, stopper.asked[0].TenantInfo.OrgID)
	assert.Equal(t, sess.BusinessUnitID, stopper.asked[0].TenantInfo.BuID)
	deps.sessionRepo.AssertExpectations(t)
}

// Stopping the replies is best effort. Signing out never fails over it.
func TestLogout_SucceedsWhenTheRepliesCouldNotBeStopped(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	stopper := &turnStopper{err: errors.New("temporal is down")}
	deps.svc.turns = stopper
	sess := signedIn()

	deps.sessionRepo.On("Get", mock.Anything, sess.ID).Return(sess, nil)
	deps.sessionRepo.On("Delete", mock.Anything, sess.ID).Return(nil)

	require.NoError(t, deps.svc.Logout(t.Context(), sess.ID))
	assert.Len(t, stopper.asked, 1)
}

// A session that cannot be read is still ended. There is only no telling
// whose replies to stop.
func TestLogout_EndsASessionItCouldNotRead(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	stopper := &turnStopper{}
	deps.svc.turns = stopper
	sessionID := pulid.MustNew("ses_")

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(nil, errors.New("not found"))
	deps.sessionRepo.On("Delete", mock.Anything, sessionID).Return(nil)

	require.NoError(t, deps.svc.Logout(t.Context(), sessionID))
	assert.Empty(t, stopper.asked)
	deps.sessionRepo.AssertExpectations(t)
}

// A session that was not ended leaves its person signed in, so their replies
// are left running too.
func TestLogout_LeavesRepliesRunningWhenTheSessionWasNotEnded(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	stopper := &turnStopper{}
	deps.svc.turns = stopper
	sess := signedIn()

	deps.sessionRepo.On("Get", mock.Anything, sess.ID).Return(sess, nil)
	deps.sessionRepo.On("Delete", mock.Anything, sess.ID).Return(errors.New("delete failed"))

	require.Error(t, deps.svc.Logout(t.Context(), sess.ID))
	assert.Empty(t, stopper.asked)
}
