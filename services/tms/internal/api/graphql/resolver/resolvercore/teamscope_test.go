package resolvercore

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scopedEngine struct {
	permissionports.PermissionEngine

	scope permission.DataScope
}

func (e *scopedEngine) Check(
	context.Context,
	*permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: true, DataScope: e.scope}, nil
}

func scopedContext(t *testing.T) context.Context {
	t.Helper()

	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func TestRequireTeamScope_NarrowsAnOwnGrantLikeATeamGrant(t *testing.T) {
	t.Parallel()

	for _, scope := range []permission.DataScope{permission.DataScopeOwn, permission.DataScopeTeam} {
		core := &Core{PermissionEngine: &scopedEngine{scope: scope}}

		_, _, err := core.RequireTeamScope(
			scopedContext(t),
			permission.ResourceWorkerPTO,
			permission.OpApprove,
			approvaldelegation.ApprovalScopeTimeOff,
			pulid.MustNew("wrk_"),
		)
		require.Error(t, err, scope)
		assert.True(t, errortypes.IsAuthorizationError(err), scope)

		_, _, err = core.RequireListTeamScope(
			scopedContext(t),
			permission.ResourceWorkerPTO,
			permission.OpRead,
			approvaldelegation.ApprovalScopeTimeOff,
		)
		require.Error(t, err, scope)
		assert.True(t, errortypes.IsAuthorizationError(err), scope)
	}
}

func TestRequireTeamScope_LeavesAnOrganizationGrantAlone(t *testing.T) {
	t.Parallel()

	core := &Core{PermissionEngine: &scopedEngine{scope: permission.DataScopeOrganization}}

	_, managers, err := core.RequireListTeamScope(
		scopedContext(t),
		permission.ResourceWorkerPTO,
		permission.OpRead,
		approvaldelegation.ApprovalScopeTimeOff,
	)

	require.NoError(t, err)
	assert.Nil(t, managers)
}

func TestRequireOrganizationScope_RefusesANarrowedGrant(t *testing.T) {
	t.Parallel()

	for _, scope := range []permission.DataScope{permission.DataScopeOwn, permission.DataScopeTeam} {
		core := &Core{PermissionEngine: &scopedEngine{scope: scope}}

		_, err := core.RequireOrganizationScope(
			scopedContext(t),
			permission.ResourceTimesheet,
			permission.OpExport,
		)

		require.Error(t, err, scope)
		assert.True(t, errortypes.IsAuthorizationError(err), scope)
	}
}
