package roleservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/scopepolicytest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func grant(resource string, scope permission.DataScope) *permission.ResourcePermission {
	return &permission.ResourcePermission{
		Resource:   resource,
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  scope,
	}
}

func scopeErrorFields(t *testing.T, err error) []string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr), "expected field errors, got %v", err)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	return fields
}

func TestCreateRole_RefusesAScopeTheKindDoesNotEnforce(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)

	err := deps.svc.CreateRole(t.Context(), CreateRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		Role: &permission.Role{
			Name:           "Customer desk",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceShipment.String(), permission.DataScopeOwn),
				grant(permission.ResourceCustomer.String(), permission.DataScopeOwn),
			},
		},
	})

	require.Error(t, err)
	assert.Equal(t, []string{"permissions[1].dataScope"}, scopeErrorFields(t, err))
	deps.roleRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreateRole_RefusesADatabaseScopeWhileTheDatabaseDoesNotEnforceIt(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.svc.scopes = &scopepolicytest.Policy{Registry: permissioncatalog.NewRegistry()}

	err := deps.svc.CreateRole(t.Context(), CreateRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		Role: &permission.Role{
			Name:           "Own loads",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceShipment.String(), permission.DataScopeOwn),
				grant(permission.ResourceCaptureBatch.String(), permission.DataScopeOwn),
			},
		},
	})

	require.Error(t, err)
	assert.Equal(t, []string{"permissions[0].dataScope"}, scopeErrorFields(t, err))
}

func TestUpdateRole_KeepsAScopeTheRoleAlreadyHeld(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	deps.svc.scopes = &scopepolicytest.Policy{Registry: permissioncatalog.NewRegistry()}
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	held := grant(permission.ResourceShipment.String(), permission.DataScopeOwn)

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: orgID,
		Permissions:    []*permission.ResourcePermission{held},
	}, nil)
	deps.roleRepo.On("Update", ctx, mock.Anything).Return(nil)

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		Role: &permission.Role{
			ID:             roleID,
			Name:           "Renamed",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceShipment.String(), permission.DataScopeOwn),
			},
		},
	})

	require.NoError(t, err)
}

func TestCreateResourcePermission_RefusesTeamOnAKindWithoutTeams(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{ID: roleID, OrganizationID: orgID}, nil)

	rp := grant(permission.ResourceInvoice.String(), permission.DataScopeTeam)
	rp.RoleID = roleID
	err := deps.svc.CreateResourcePermission(ctx, pulid.MustNew("usr_"), orgID, rp)

	require.Error(t, err)
	assert.Equal(t, []string{"dataScope"}, scopeErrorFields(t, err))
	deps.roleRepo.AssertNotCalled(t, "CreateResourcePermission", mock.Anything, mock.Anything)
}

type activationPolicy bool

func (p activationPolicy) RoleActivationRequired(
	context.Context,
	pagination.TenantInfo,
) (bool, error) {
	return bool(p), nil
}

func TestSaveRoleConstraint_DynamicSeparationNeedsRoleActivation(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.svc.activation = activationPolicy(false)
	orgID := pulid.MustNew("org_")
	first, second := pulid.MustNew("rol_"), pulid.MustNew("rol_")
	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{}, nil)

	err := deps.svc.SaveRoleConstraint(t.Context(), &SaveConstraintRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
		Constraint: &permission.RoleConstraint{
			Name:     "Approve or post, not both at once",
			Type:     permission.RoleConstraintTypeDSD,
			MaxRoles: 1,
		},
		RoleIDs: []pulid.ID{first, second},
	})

	var fieldErr *errortypes.Error
	require.True(t, errors.As(err, &fieldErr), "expected a field error, got %v", err)
	assert.Equal(t, "type", fieldErr.Field)
}
