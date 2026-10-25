package rolerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRolesWithInheritanceIsADedupingRecursiveCTE(t *testing.T) {
	t.Parallel()

	roleID := pulid.ID("rol_1")
	sql, args := keeltest.Render(t, rolesWithInheritance(keeltest.TenantContext(t), []pulid.ID{roleID}))

	assert.Contains(t, sql, "WITH RECURSIVE")
	assert.Contains(t, sql, `"role_closure"`)
	assert.Contains(t, sql, "= ANY(")
	assert.Contains(t, sql, " UNION ")
	assert.NotContains(t, sql, "UNION ALL")
	assert.NotContains(t, sql, "resource_permissions", "permissions load in their own statement")
	assert.Contains(t, sql, `"organization_id" = `)
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
}

func TestRolesWithInheritanceCrossesTenantsOnlyUnderASystemScope(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "role closure test")
	sql, args := keeltest.RenderIn(t, ctx, rolesWithInheritance(ctx, []pulid.ID{"rol_1"}))

	assert.Contains(t, sql, "WITH RECURSIVE")
	assert.NotContains(t, args, string(keeltest.TestOrganizationID))
}

func TestGetRolesWithInheritanceSkipsTheQueryForAnEmptyRoleSet(t *testing.T) {
	t.Parallel()

	repo := &repository{db: postgres.NewTestConnection(keeltest.RenderDB(t)), l: zap.NewNop()}

	roles, err := repo.GetRolesWithInheritance(keeltest.TenantContext(t), nil)
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestOrphanedAssignmentsQueryCrossesTenants(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "orphaned assignments test")
	keeltest.SnapshotIn(t, ctx, orphanedAssignmentsQuery(100, 50))
}

func TestUsersWithPermissionQueryWalksParentRoles(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, usersWithPermissionQuery(&permissionports.ListUsersWithPermissionRequest{
		Resource:  permission.ResourceAgentProposal,
		Operation: permission.OpUpdate,
		Now:       100,
	}))
}
