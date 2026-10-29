//go:build integration

package rolerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRoleResolutionIsServedFromTheResultCacheUntilARoleChanges(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	cached := cachetest.Open(t, ctx, dsn)
	repo := New(Params{DB: postgres.NewTestConnection(cached), Logger: zap.NewNop()})
	tctx := data.Ctx(ctx)

	parent := newTestRole("Cached Parent", nil, "shipment")
	require.NoError(t, repo.Create(tctx, parent))
	child := newTestRole("Cached Child", []pulid.ID{parent.ID}, "customer")
	require.NoError(t, repo.Create(tctx, child))
	require.NoError(t, repo.CreateAssignment(tctx, &rbac.UserRoleAssignment{
		UserID:         data.User.ID,
		OrganizationID: data.Organization.ID,
		RoleID:         child.ID,
		AssignedBy:     data.User.ID,
	}))

	var assignments []*rbac.UserRoleAssignment
	readAssignments := func() error {
		var readErr error
		assignments, readErr = repo.GetUserRoleAssignments(
			tctx,
			data.User.ID,
			data.Organization.ID,
		)
		return readErr
	}
	var roles []*rbac.Role
	readRoles := func() error {
		var readErr error
		roles, readErr = repo.GetRolesWithInheritance(tctx, []pulid.ID{child.ID})
		return readErr
	}

	t.Run("a second read is a hit with the same rows", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readAssignments)
		cachetest.RequireHit(t, cached, readAssignments)
		require.Len(t, assignments, 1)
		require.NotNil(t, assignments[0].Role, "the role load is replayed from the entry")
		assert.Equal(t, child.ID, assignments[0].Role.ID)

		cachetest.RequireMiss(t, cached, readRoles)
		cachetest.RequireHit(t, cached, readRoles)
		assert.Len(t, roles, 2)
		for _, role := range roles {
			assert.Len(t, role.Permissions, 1, "role %s keeps its permissions on a hit", role.Name)
		}
	})

	t.Run("a new assignment invalidates the user's assignments", func(t *testing.T) {
		require.NoError(t, repo.CreateAssignment(tctx, &rbac.UserRoleAssignment{
			UserID:         data.User.ID,
			OrganizationID: data.Organization.ID,
			RoleID:         parent.ID,
			AssignedBy:     data.User.ID,
		}))

		cachetest.RequireMiss(t, cached, readAssignments)
		assert.Len(t, assignments, 2)
		cachetest.RequireHit(t, cached, readAssignments)
		assert.Len(t, assignments, 2)
	})

	t.Run("changing a parent role's permissions invalidates the closure", func(t *testing.T) {
		parent.Permissions = []*rbac.ResourcePermission{
			{
				Resource:   "shipment",
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
			},
			{
				Resource:   "invoice",
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
			},
		}
		require.NoError(t, repo.Update(tctx, parent))

		cachetest.RequireMiss(t, cached, readRoles)
		byName := make(map[string]*rbac.Role, len(roles))
		for _, role := range roles {
			byName[role.Name] = role
		}
		require.Contains(t, byName, "Cached Parent")
		assert.Len(t, byName["Cached Parent"].Permissions, 2)
		cachetest.RequireHit(t, cached, readRoles)
	})
}
