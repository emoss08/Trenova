package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubDelegated struct {
	userID pulid.ID
	perms  *permissionports.CachedPermissions
	err    error
}

func (s *stubDelegated) DelegatedPermissions(
	_ context.Context,
	userID, _ pulid.ID,
) (*permissionports.CachedPermissions, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	if userID != s.userID {
		return nil, false, nil
	}
	return s.perms, true, nil
}

func TestCheckUsesADelegatedPermissionSetWithoutTouchingRoles(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	eng.delegated = &stubDelegated{
		userID: userID,
		perms: &permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityConfidential),
			Resources: map[string]*permissionports.CachedResourcePermission{
				permission.ResourceShipment.String(): {
					Operations: []string{string(permission.OpRead)},
					DataScope:  string(permission.DataScopeOrganization),
				},
			},
		},
	}

	read, err := eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       permission.ResourceShipment.String(),
		Operation:      permission.OpRead,
	})
	require.NoError(t, err)
	assert.True(t, read.Allowed)

	update, err := eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       permission.ResourceShipment.String(),
		Operation:      permission.OpUpdate,
	})
	require.NoError(t, err)
	assert.False(t, update.Allowed)
}

func TestCheckFallsBackToRolesWhenTheSourceDeclines(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	eng.delegated = &stubDelegated{userID: pulid.MustNew("usr_")}

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).Return(nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       permission.ResourceShipment.String(),
		Operation:      permission.OpRead,
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestCheckFailsClosedWhenTheSourceErrors(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	eng.delegated = &stubDelegated{err: errors.New("session store unavailable")}

	_, err := eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		Resource:       permission.ResourceShipment.String(),
		Operation:      permission.OpRead,
	})
	require.Error(t, err)
}
