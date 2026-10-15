package userservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (d *testDeps) expectUserManagement(actorID, orgID pulid.ID, allowed bool) {
	d.permissions.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
			return req.OrganizationID == orgID && req.UserID == actorID
		})).
		Return(&permissionports.PermissionCheckResult{Allowed: allowed}, nil).
		Once()
}

func TestReplaceOrganizationMemberships_RefusesAnOrganizationTheActorCannotManage(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.roles = &fakeRoles{log: &eventLog{}}

	actorID, userID := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	current, other := pulid.MustNew("org_"), pulid.MustNew("org_")

	deps.repo.EXPECT().ListOrganizationMemberships(mock.Anything, userID, buID).
		Return(memberships(current, other), nil).
		Once()
	deps.expectUserManagement(actorID, other, false)

	_, err := deps.svc.ReplaceOrganizationMemberships(
		t.Context(),
		actorID,
		userID,
		current,
		buID,
		[]pulid.ID{current},
	)

	require.Error(t, err)
	var authErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authErr)
	deps.repo.AssertNotCalled(t, "ReplaceOrganizationMemberships", mock.Anything, mock.Anything)
}

func TestReplaceOrganizationMemberships_RefusesAddingAnOrganizationTheActorCannotManage(
	t *testing.T,
) {
	t.Parallel()

	deps := setupTest(t)

	actorID, userID := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	current, other := pulid.MustNew("org_"), pulid.MustNew("org_")

	deps.repo.EXPECT().ListOrganizationMemberships(mock.Anything, userID, buID).
		Return(memberships(current), nil).
		Once()
	deps.expectUserManagement(actorID, other, false)

	_, err := deps.svc.ReplaceOrganizationMemberships(
		t.Context(),
		actorID,
		userID,
		current,
		buID,
		[]pulid.ID{current, other},
	)

	require.Error(t, err)
	deps.repo.AssertNotCalled(t, "ReplaceOrganizationMemberships", mock.Anything, mock.Anything)
}

func TestReplaceOrganizationMemberships_RefusesRemovingEveryOrganization(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)

	_, err := deps.svc.ReplaceOrganizationMemberships(
		t.Context(),
		pulid.MustNew("usr_"),
		pulid.MustNew("usr_"),
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
		nil,
	)

	require.Error(t, err)
	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "organizationIds", validationErr.Field)
	deps.repo.AssertNotCalled(t, "ListOrganizationMemberships", mock.Anything, mock.Anything, mock.Anything)
}

func TestMayManageUsersIn_ChecksTheOtherOrganizationsOwnRoles(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	actorID, orgID, buID := pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_")
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{pulid.MustNew("rol_")}, true)

	deps.permissions.EXPECT().
		Check(mock.Anything, mock.Anything).
		RunAndReturn(func(
			checkCtx context.Context,
			req *permissionports.PermissionCheckRequest,
		) (*permissionports.PermissionCheckResult, error) {
			_, activated := authctx.GetSessionRoleActivation(checkCtx)
			assert.False(t, activated, "the session's activated roles belong to another organization")
			tenant, scoped := dbscope.TenantFrom(checkCtx)
			require.True(t, scoped)
			assert.Equal(t, orgID, tenant.OrganizationID)
			assert.Equal(t, buID, tenant.BusinessUnitID)
			assert.Equal(t, permission.ResourceUser.String(), req.Resource)
			assert.Equal(t, permission.OpUpdate, req.Operation)
			assert.Equal(t, orgID, req.OrganizationID)
			return &permissionports.PermissionCheckResult{Allowed: true}, nil
		}).
		Once()

	allowed, err := deps.svc.mayManageUsersIn(ctx, actorID, orgID, buID)

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestChangedOrganizations(t *testing.T) {
	t.Parallel()

	kept, removed, added := pulid.MustNew("org_"), pulid.MustNew("org_"), pulid.MustNew("org_")

	changed := changedOrganizations(memberships(kept, removed), []pulid.ID{kept, added, added})

	assert.ElementsMatch(t, []pulid.ID{removed, added}, changed)
}
