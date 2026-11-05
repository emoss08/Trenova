package permission

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck_ANarrowGrantHoldsOnlyWhereItsScopeIsEnforced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		resource  permission.Resource
		scope     permission.DataScope
		enforcing bool
		allowed   bool
	}{
		{
			name:      "own on a database kind, database enforcing",
			resource:  permission.ResourceShipment,
			scope:     permission.DataScopeOwn,
			enforcing: true,
			allowed:   true,
		},
		{
			name:     "own on a database kind, database not enforcing",
			resource: permission.ResourceShipment,
			scope:    permission.DataScopeOwn,
		},
		{
			name:     "team on a database kind, database not enforcing",
			resource: permission.ResourceWorker,
			scope:    permission.DataScopeTeam,
		},
		{
			name:     "organization on a database kind, database not enforcing",
			resource: permission.ResourceShipment,
			scope:    permission.DataScopeOrganization,
			allowed:  true,
		},
		{
			name:     "own on an application kind, database not enforcing",
			resource: permission.ResourceReport,
			scope:    permission.DataScopeOwn,
			allowed:  true,
		},
		{
			name:     "team on an application kind, database not enforcing",
			resource: permission.ResourceTimesheet,
			scope:    permission.DataScopeTeam,
			allowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, userID, orgID := cachedEngine(
				t,
				map[string]*permissionports.CachedResourcePermission{
					tt.resource.String(): cachedGrant(tt.scope, permission.OpRead),
				},
			)
			eng.databaseEnforcing = tt.enforcing

			result, err := eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
				UserID:         userID,
				OrganizationID: orgID,
				Resource:       tt.resource.String(),
				Operation:      permission.OpRead,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, result.Allowed)
		})
	}
}

func TestResolveResourcePermission_AnUnenforcedScopeCoversTheKindsUnderIt(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	eng.databaseEnforcing = false
	worker := permission.ResourceWorker.String()
	child := permission.ResourceWorkerCredential.String()
	require.Equal(t, worker, eng.registry.GetEffectiveResource(child))

	_, ok := eng.resolveResourcePermission(
		&permissionports.CachedPermissions{
			Resources: map[string]*permissionports.CachedResourcePermission{
				worker: cachedGrant(permission.DataScopeTeam, permission.OpRead),
			},
		},
		child,
		permission.OpRead,
	)
	assert.False(t, ok)
}

func TestAPIKeyCheck_AnUnenforcedScopeAllowsNothing(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	eng.databaseEnforcing = false
	shipment := permission.ResourceShipment.String()

	result := eng.resolveAPIKeyPermission(
		&permissionports.CachedPermissions{
			Resources: map[string]*permissionports.CachedResourcePermission{
				shipment: cachedGrant(permission.DataScopeOwn, permission.OpRead),
			},
		},
		&permissionports.PermissionCheckRequest{
			OrganizationID: pulid.MustNew("org_"),
			Resource:       shipment,
			Operation:      permission.OpRead,
		},
		pulid.MustNew("usr_"),
		time.Now(),
	)
	assert.False(t, result.Allowed)
}
