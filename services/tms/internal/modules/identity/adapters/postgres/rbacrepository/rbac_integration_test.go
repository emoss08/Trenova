//go:build integration

package rbacrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCountRolePermissions_FollowsInheritedRolesAndCountsEachOperation(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	conn := postgres.NewTestConnection(db)
	create := func(role *rbac.Role) {
		t.Helper()
		require.NoError(t, rbacdb.Roles.Create(ctx, db, role))
		for _, p := range role.Permissions {
			p.RoleID = role.ID
		}
		if len(role.Permissions) > 0 {
			require.NoError(
				t,
				rbacdb.ResourcePermissions.CreateMany(ctx, db, role.Permissions),
			)
		}
	}
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	grant := func(resource string, ops ...permission.Operation) *rbac.ResourcePermission {
		return &rbac.ResourcePermission{
			Resource:   resource,
			Operations: ops,
			DataScope:  permission.DataScopeOrganization,
		}
	}
	parent := &rbac.Role{
		Name:           "Dispatch lead",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions: []*rbac.ResourcePermission{
			grant("shipment", permission.OpRead, permission.OpUpdate),
			grant("driver", permission.OpRead),
		},
	}
	create(parent)
	child := &rbac.Role{
		Name:           "Dispatcher",
		ParentRoleIDs:  []pulid.ID{parent.ID},
		MaxSensitivity: permission.SensitivityInternal,
		Permissions: []*rbac.ResourcePermission{
			grant("worker", permission.OpRead),
			grant("shipment", permission.OpRead),
		},
	}
	create(child)
	empty := &rbac.Role{Name: "Observer", MaxSensitivity: permission.SensitivityInternal}
	create(empty)

	counts, err := repo.CountRolePermissions(ctx, []pulid.ID{child.ID, parent.ID, empty.ID})
	require.NoError(t, err)
	assert.Equal(t, 4, counts[child.ID])
	assert.Equal(t, 3, counts[parent.ID])
	assert.Equal(t, 0, counts[empty.ID])
}
