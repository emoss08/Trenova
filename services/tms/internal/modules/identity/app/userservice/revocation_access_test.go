package userservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingRevoker struct {
	users []pulid.ID
	err   error
}

func (r *recordingRevoker) RevokeUserAccess(
	_ context.Context,
	req tenantports.RevokeUserAccessRequest,
) error {
	r.users = append(r.users, req.UserID)

	return r.err
}

func TestEndAllSessionsRevokesLongLivedAccessToo(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.authEvents = &recordedEvents{}
	revoker := &recordingRevoker{}
	deps.svc.accessRevokers = []tenantports.UserAccessRevoker{revoker}
	userID := pulid.MustNew("usr_")
	deps.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, userID).Return(nil).Once()

	require.NoError(t, deps.svc.endAllSessions(t.Context(), pagination.TenantInfo{}, userID))

	assert.Equal(t, []pulid.ID{userID}, revoker.users)
}

func TestEndAllSessionsRevokesAccessEvenWhenSessionsCouldNotBeEnded(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.authEvents = &recordedEvents{}
	revoker := &recordingRevoker{}
	deps.svc.accessRevokers = []tenantports.UserAccessRevoker{revoker}
	userID := pulid.MustNew("usr_")
	deps.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, userID).
		Return(errors.New("redis unavailable")).
		Once()

	require.Error(t, deps.svc.endAllSessions(t.Context(), pagination.TenantInfo{}, userID))

	assert.Equal(t, []pulid.ID{userID}, revoker.users)
}

func TestEndAllSessionsReportsAFailedRevocation(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.authEvents = &recordedEvents{}
	failure := errors.New("database unavailable")
	deps.svc.accessRevokers = []tenantports.UserAccessRevoker{&recordingRevoker{err: failure}}
	userID := pulid.MustNew("usr_")
	deps.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, userID).Return(nil).Once()

	err := deps.svc.endAllSessions(t.Context(), pagination.TenantInfo{}, userID)

	require.ErrorIs(t, err, failure)
}
