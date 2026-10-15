package userservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func orphan() permissionports.OrphanedAssignment {
	return permissionports.OrphanedAssignment{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestSweepOrphanedOrganizationAccess_RevokesRolesAndSessions(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	roles := &fakeRoles{log: &eventLog{}}
	deps.svc.roles = roles
	left := orphan()
	stale := &session.Session{ID: pulid.MustNew("ses_"), OrganizationID: left.OrganizationID}
	elsewhere := &session.Session{ID: pulid.MustNew("ses_"), OrganizationID: pulid.MustNew("org_")}

	deps.roleRepo.EXPECT().ListOrphanedAssignments(mock.Anything, int64(100), orphanSweepBatchSize).
		Return([]permissionports.OrphanedAssignment{left}, nil).Once()
	deps.sessionRepo.EXPECT().ListForUser(mock.Anything, left.UserID).
		Return([]*session.Session{stale, elsewhere}, nil).Once()
	deps.sessionRepo.EXPECT().Delete(mock.Anything, stale.ID).Return(nil).Once()

	result, err := deps.svc.SweepOrphanedOrganizationAccess(t.Context(), 100)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Revoked)
	require.Len(t, roles.revoked, 1)
	assert.Equal(t, left.UserID, roles.revoked[0].UserID)
	assert.Equal(t, left.OrganizationID, roles.revoked[0].OrganizationID)
	assert.Equal(t, left.BusinessUnitID, roles.revoked[0].BusinessUnitID)
	assert.True(t, roles.revoked[0].ActorID.IsNil())
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, elsewhere.ID)
}

func TestSweepOrphanedOrganizationAccess_ReadsAnotherBatchWhenOneIsFull(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.roles = &fakeRoles{log: &eventLog{}}

	full := make([]permissionports.OrphanedAssignment, orphanSweepBatchSize)
	for i := range full {
		full[i] = orphan()
	}
	last := orphan()

	deps.roleRepo.EXPECT().ListOrphanedAssignments(mock.Anything, int64(100), orphanSweepBatchSize).
		Return(full, nil).Once()
	deps.roleRepo.EXPECT().ListOrphanedAssignments(mock.Anything, int64(100), orphanSweepBatchSize).
		Return([]permissionports.OrphanedAssignment{last}, nil).Once()
	deps.sessionRepo.EXPECT().ListForUser(mock.Anything, mock.Anything).Return(nil, nil)

	result, err := deps.svc.SweepOrphanedOrganizationAccess(t.Context(), 100)

	require.NoError(t, err)
	assert.Equal(t, orphanSweepBatchSize+1, result.Revoked)
}

func TestSweepOrphanedOrganizationAccess_StopsWhenABatchHasFailures(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.roles = &fakeRoles{log: &eventLog{}, revokeErr: errors.New("database unavailable")}

	full := make([]permissionports.OrphanedAssignment, orphanSweepBatchSize)
	for i := range full {
		full[i] = orphan()
	}

	deps.roleRepo.EXPECT().ListOrphanedAssignments(mock.Anything, int64(100), orphanSweepBatchSize).
		Return(full, nil).Once()

	result, err := deps.svc.SweepOrphanedOrganizationAccess(t.Context(), 100)

	require.Error(t, err)
	assert.Equal(t, orphanSweepBatchSize, result.Failed)
	assert.Zero(t, result.Revoked)
}

func TestSweepOrphanedOrganizationAccess_ReportsAFailedListing(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.roleRepo.EXPECT().ListOrphanedAssignments(mock.Anything, int64(100), orphanSweepBatchSize).
		Return(nil, errors.New("database unavailable")).Once()

	_, err := deps.svc.SweepOrphanedOrganizationAccess(t.Context(), 100)

	require.Error(t, err)
}
