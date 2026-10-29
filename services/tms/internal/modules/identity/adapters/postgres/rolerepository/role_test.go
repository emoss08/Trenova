//go:build integration

package rolerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type roleFixture struct {
	ctx  context.Context
	db   *keel.DB
	repo permissionports.RoleRepository
	data *seedtest.TestData
}

func newRoleFixture(t *testing.T) *roleFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)

	return &roleFixture{
		ctx:  data.Ctx(ctx),
		db:   db,
		repo: New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}),
		data: data,
	}
}

func newTestRole(name string, parents []pulid.ID, resources ...string) *rbac.Role {
	role := &rbac.Role{
		Name:           name,
		ParentRoleIDs:  parents,
		MaxSensitivity: permission.SensitivityInternal,
	}
	for _, resource := range resources {
		role.Permissions = append(role.Permissions, &rbac.ResourcePermission{
			Resource:   resource,
			Operations: []permission.Operation{permission.OpRead},
			DataScope:  permission.DataScopeOrganization,
		})
	}

	return role
}

func roleNameSet(roles []*rbac.Role) map[string]bool {
	names := make(map[string]bool, len(roles))
	for _, role := range roles {
		names[role.Name] = true
	}
	return names
}

func TestRoleRepository_CreateAndGetByID(t *testing.T) {
	f := newRoleFixture(t)

	role := newTestRole("Dispatcher", nil, "shipment", "driver")
	role.Description = "Plans loads"
	require.NoError(t, f.repo.Create(f.ctx, role))
	require.True(t, role.ID.IsNotNil())
	assert.Equal(t, f.data.Organization.ID, role.OrganizationID)
	assert.Equal(t, f.data.BusinessUnit.ID, role.BusinessUnitID)

	loaded, err := f.repo.GetByID(f.ctx, permissionports.GetRoleByIDRequest{
		ID:         role.ID,
		TenantInfo: f.data.Info(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Dispatcher", loaded.Name)
	assert.Equal(t, "Plans loads", loaded.Description)
	assert.Len(t, loaded.Permissions, 2)

	_, err = f.repo.GetByID(f.ctx, permissionports.GetRoleByIDRequest{
		ID:         pulid.MustNew("rol_"),
		TenantInfo: f.data.Info(),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestRoleRepository_ListAndUpdate(t *testing.T) {
	f := newRoleFixture(t)

	for _, name := range []string{"Billing Clerk", "Dispatcher", "Safety Manager"} {
		require.NoError(t, f.repo.Create(f.ctx, newTestRole(name, nil, "shipment")))
	}
	other := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "RL")
	require.NoError(t, f.repo.Create(other.Ctx(f.ctx), newTestRole("Elsewhere", nil, "shipment")))

	listed, err := f.repo.List(f.ctx, &permissionports.ListRolesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.data.Info(),
			Pagination: pagination.Info{Limit: 2},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, listed.Total)
	assert.Len(t, listed.Items, 2)
	assert.False(t, roleNameSet(listed.Items)["Elsewhere"])

	role := listed.Items[0]
	role.Name = "Renamed"
	role.Permissions = []*rbac.ResourcePermission{{
		Resource:   "customer",
		Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:  permission.DataScopeOrganization,
	}}
	require.NoError(t, f.repo.Update(f.ctx, role))

	loaded, err := f.repo.GetByID(f.ctx, permissionports.GetRoleByIDRequest{
		ID:         role.ID,
		TenantInfo: f.data.Info(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", loaded.Name)
	require.Len(t, loaded.Permissions, 1)
	assert.Equal(t, "customer", loaded.Permissions[0].Resource)
}

func TestRoleRepository_RoleAssignments(t *testing.T) {
	f := newRoleFixture(t)

	role := newTestRole("Assignment Role", nil, "shipment")
	require.NoError(t, f.repo.Create(f.ctx, role))

	assignment := &rbac.UserRoleAssignment{
		UserID:         f.data.User.ID,
		OrganizationID: f.data.Organization.ID,
		RoleID:         role.ID,
		AssignedBy:     f.data.User.ID,
	}
	require.NoError(t, f.repo.CreateAssignment(f.ctx, assignment))
	require.True(t, assignment.ID.IsNotNil())

	err := f.repo.CreateAssignment(f.ctx, &rbac.UserRoleAssignment{
		UserID:         f.data.User.ID,
		OrganizationID: f.data.Organization.ID,
		RoleID:         role.ID,
	})
	require.Error(t, err, "a second active assignment of the same role is refused")

	assignments, err := f.repo.GetUserRoleAssignments(f.ctx, f.data.User.ID, f.data.Organization.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.NotNil(t, assignments[0].Role)
	assert.Equal(t, role.ID, assignments[0].Role.ID)

	users, err := f.repo.GetUsersWithRole(f.ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, f.data.User.ID, users[0].UserID)

	deleted, err := f.repo.DeleteAssignment(f.ctx, permissionports.DeleteRoleAssignmentRequest{
		AssignmentID:   assignment.ID,
		OrganizationID: f.data.Organization.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, f.data.User.ID, deleted.UserID)

	assignments, err = f.repo.GetUserRoleAssignments(f.ctx, f.data.User.ID, f.data.Organization.ID)
	require.NoError(t, err)
	assert.Empty(t, assignments)
}

func TestRoleRepository_GetRolesWithInheritance(t *testing.T) {
	f := newRoleFixture(t)

	grandparent := newTestRole("Grandparent", nil, "shipment")
	require.NoError(t, f.repo.Create(f.ctx, grandparent))
	parent := newTestRole("Parent", []pulid.ID{grandparent.ID}, "driver")
	require.NoError(t, f.repo.Create(f.ctx, parent))
	child := newTestRole("Child", []pulid.ID{parent.ID}, "customer")
	require.NoError(t, f.repo.Create(f.ctx, child))
	unrelated := newTestRole("Unrelated", nil, "invoice")
	require.NoError(t, f.repo.Create(f.ctx, unrelated))

	t.Run("a three deep chain resolves in full", func(t *testing.T) {
		roles, err := f.repo.GetRolesWithInheritance(f.ctx, []pulid.ID{child.ID})
		require.NoError(t, err)

		names := roleNameSet(roles)
		assert.Len(t, roles, 3)
		assert.True(t, names["Child"])
		assert.True(t, names["Parent"])
		assert.True(t, names["Grandparent"])
		assert.False(t, names["Unrelated"])
	})

	t.Run("permissions are loaded for inherited roles", func(t *testing.T) {
		roles, err := f.repo.GetRolesWithInheritance(f.ctx, []pulid.ID{child.ID})
		require.NoError(t, err)

		resources := make(map[string]bool, len(roles))
		for _, role := range roles {
			require.Len(t, role.Permissions, 1, "role %s lost its permissions", role.Name)
			resources[role.Permissions[0].Resource] = true
		}
		assert.True(t, resources["shipment"])
		assert.True(t, resources["driver"])
		assert.True(t, resources["customer"])
	})

	t.Run("a role reached by two paths is returned once", func(t *testing.T) {
		diamond := newTestRole("Diamond", []pulid.ID{parent.ID, grandparent.ID}, "worker")
		require.NoError(t, f.repo.Create(f.ctx, diamond))

		roles, err := f.repo.GetRolesWithInheritance(f.ctx, []pulid.ID{diamond.ID})
		require.NoError(t, err)
		assert.Len(t, roles, 3)
	})

	t.Run("several roots resolve together", func(t *testing.T) {
		roles, err := f.repo.GetRolesWithInheritance(f.ctx, []pulid.ID{child.ID, unrelated.ID})
		require.NoError(t, err)

		names := roleNameSet(roles)
		assert.True(t, names["Child"])
		assert.True(t, names["Grandparent"])
		assert.True(t, names["Unrelated"])
	})

	t.Run("an empty role set returns nothing", func(t *testing.T) {
		roles, err := f.repo.GetRolesWithInheritance(f.ctx, []pulid.ID{})
		require.NoError(t, err)
		assert.Empty(t, roles)
	})

	t.Run("another tenant's roles are never reached", func(t *testing.T) {
		other := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "RI")
		roles, err := f.repo.GetRolesWithInheritance(other.Ctx(f.ctx), []pulid.ID{child.ID})
		require.NoError(t, err)
		assert.Empty(t, roles)
	})
}

func TestRoleRepository_GetRolesWithInheritanceStopsAtACycle(t *testing.T) {
	f := newRoleFixture(t)

	first := newTestRole("Cycle First", nil, "shipment")
	require.NoError(t, f.repo.Create(f.ctx, first))
	second := newTestRole("Cycle Second", []pulid.ID{first.ID}, "driver")
	require.NoError(t, f.repo.Create(f.ctx, second))

	ro := permissiondb.Roles
	_, err := ro.Where(ro.ID.Eq(first.ID)).
		Update(f.ctx, f.db, ro.ParentRoleIDs.To([]string{second.ID.String()}))
	require.NoError(t, err)

	roles, err := f.repo.GetRolesWithInheritance(f.ctx, []pulid.ID{second.ID})
	require.NoError(t, err)

	names := roleNameSet(roles)
	assert.Len(t, roles, 2)
	assert.True(t, names["Cycle First"])
	assert.True(t, names["Cycle Second"])
}

func TestRoleRepository_ResourcePermissions(t *testing.T) {
	f := newRoleFixture(t)

	role := newTestRole("Permission Test Role", nil)
	require.NoError(t, f.repo.Create(f.ctx, role))

	rp := &rbac.ResourcePermission{
		RoleID:     role.ID,
		Resource:   "customer",
		Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:  permission.DataScopeOrganization,
	}
	require.NoError(t, f.repo.CreateResourcePermission(f.ctx, rp))

	permissions, err := f.repo.GetResourcePermissionsByRoleID(f.ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, permissions, 1)
	assert.Equal(t, "customer", permissions[0].Resource)

	rp.Operations = append(rp.Operations, permission.OpExport)
	require.NoError(t, f.repo.UpdateResourcePermission(f.ctx, rp))

	permissions, err = f.repo.GetResourcePermissionsByRoleID(f.ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, permissions, 1)
	assert.Contains(t, permissions[0].Operations, permission.OpExport)

	require.NoError(t, f.repo.DeleteResourcePermission(f.ctx,
		permissionports.DeleteResourcePermissionRequest{PermissionID: rp.ID, RoleID: role.ID}))

	permissions, err = f.repo.GetResourcePermissionsByRoleID(f.ctx, role.ID)
	require.NoError(t, err)
	assert.Empty(t, permissions)
}

func systemCtx(ctx context.Context) context.Context {
	return dbscope.WithSystem(ctx, "role repository test")
}
