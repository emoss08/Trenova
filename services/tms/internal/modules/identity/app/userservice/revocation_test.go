package userservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPatch_DeactivatingAUserEndsTheirSessions(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.target.IsLocked = false
	d.expectLoad(d.target)
	d.expectSave()
	d.expectAuditLog()
	d.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, d.target.ID).Return(nil).Once()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.Status = domaintypes.StatusInactive
			return nil
		},
	})

	require.NoError(t, err)
}

func TestPatch_LockingAUserEndsTheirSessions(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.target.IsLocked = false
	d.expectLoad(d.target)
	d.expectSave()
	d.expectAuditLog()
	d.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, d.target.ID).Return(nil).Once()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.IsLocked = true
			return nil
		},
	})

	require.NoError(t, err)
}

func TestPatch_ReportsSessionsThatCouldNotBeEnded(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.target.IsLocked = false
	d.expectLoad(d.target)
	d.expectSave()
	d.expectAuditLog()
	d.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, d.target.ID).
		Return(errors.New("redis unavailable")).Once()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.Status = domaintypes.StatusInactive
			return nil
		},
	})

	require.Error(t, err)
}

func TestPatch_KeepsSessionsWhenAccessIsUnchanged(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.target.IsLocked = false
	d.expectLoad(d.target)
	d.expectSave()
	d.expectAuditLog()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.JobTitle = "Dispatch lead"
			return nil
		},
	})

	require.NoError(t, err)
	d.sessionRepo.AssertNotCalled(t, "DeleteAllForUser", mock.Anything, mock.Anything)
}

func TestReplaceOrganizationMemberships_EndsOnlyTheRemovedOrganizationsSessions(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	roles := &fakeRoles{log: &eventLog{}}
	deps.svc.roles = roles

	actorID, userID := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	kept, removed := pulid.MustNew("org_"), pulid.MustNew("org_")
	keptSession := &session.Session{ID: pulid.MustNew("ses_"), OrganizationID: kept}
	removedSession := &session.Session{ID: pulid.MustNew("ses_"), OrganizationID: removed}

	deps.repo.EXPECT().ListOrganizationMemberships(mock.Anything, userID, buID).
		Return(memberships(kept, removed), nil).
		Once()
	replace := tenantports.ReplaceOrganizationMembershipsRequest{
		ActorID:         actorID,
		UserID:          userID,
		BusinessUnitID:  buID,
		OrganizationIDs: []pulid.ID{kept},
	}
	deps.repo.EXPECT().ReplaceOrganizationMemberships(mock.Anything, replace).
		Return(memberships(kept), nil).
		Once()
	deps.sessionRepo.EXPECT().ListForUser(mock.Anything, userID).
		Return([]*session.Session{keptSession, removedSession}, nil).Once()
	deps.sessionRepo.EXPECT().Delete(mock.Anything, removedSession.ID).Return(nil).Once()

	_, err := deps.svc.ReplaceOrganizationMemberships(
		t.Context(),
		actorID,
		userID,
		kept,
		buID,
		[]pulid.ID{kept},
	)

	require.NoError(t, err)
	require.Len(t, roles.revoked, 1)
	assert.Equal(t, removed, roles.revoked[0].OrganizationID)
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, keptSession.ID)
}

func TestReplaceOrganizationMemberships_ReportsRolesThatCouldNotBeRevoked(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.roles = &fakeRoles{log: &eventLog{}, revokeErr: errors.New("database unavailable")}

	actorID, userID := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	kept, removed := pulid.MustNew("org_"), pulid.MustNew("org_")

	deps.repo.EXPECT().ListOrganizationMemberships(mock.Anything, userID, buID).
		Return(memberships(kept, removed), nil).
		Once()
	deps.repo.EXPECT().ReplaceOrganizationMemberships(mock.Anything, mock.Anything).
		Return(memberships(kept), nil).
		Once()
	deps.sessionRepo.EXPECT().ListForUser(mock.Anything, userID).Return(nil, nil).Once()

	_, err := deps.svc.ReplaceOrganizationMemberships(
		t.Context(),
		actorID,
		userID,
		kept,
		buID,
		[]pulid.ID{kept},
	)

	require.Error(t, err)
}

func memberships(orgIDs ...pulid.ID) []*tenant.OrganizationMembership {
	out := make([]*tenant.OrganizationMembership, 0, len(orgIDs))
	for _, orgID := range orgIDs {
		out = append(out, &tenant.OrganizationMembership{OrganizationID: orgID})
	}
	return out
}
