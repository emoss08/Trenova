//go:build integration

package rolerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type roleTenantFixture struct {
	role       *permission.Role
	permission *permission.ResourcePermission
	assignment *permission.UserRoleAssignment
}

func seedRoleTenantFixture(
	t *testing.T,
	db *bun.DB,
	tenant *seedtest.TestData,
) roleTenantFixture {
	t.Helper()

	ctx := t.Context()
	now := timeutils.NowUnix()

	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		BusinessUnitID: tenant.BusinessUnit.ID,
		OrganizationID: tenant.Organization.ID,
		Name:           "Dispatcher " + tenant.Organization.ID.String(),
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_, err := db.NewInsert().Model(role).Exec(ctx)
	require.NoError(t, err)

	rp := &permission.ResourcePermission{
		ID:         pulid.MustNew("rp_"),
		RoleID:     role.ID,
		Resource:   permission.ResourceShipment.String(),
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err = db.NewInsert().Model(rp).Exec(ctx)
	require.NoError(t, err)

	assignment := &permission.UserRoleAssignment{
		ID:             pulid.MustNew("ura_"),
		UserID:         tenant.User.ID,
		OrganizationID: tenant.Organization.ID,
		RoleID:         role.ID,
		AssignedAt:     now,
	}
	_, err = db.NewInsert().Model(assignment).Exec(ctx)
	require.NoError(t, err)

	return roleTenantFixture{role: role, permission: rp, assignment: assignment}
}

func TestRoleRepository_DeletesAndUpdatesStayInsideTheTenant(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "RB")
	fixtureA := seedRoleTenantFixture(t, db, tenantA)
	fixtureB := seedRoleTenantFixture(t, db, tenantB)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	t.Run("an assignment cannot be deleted from another organization", func(t *testing.T) {
		_, err := repo.DeleteAssignment(ctx, repositories.DeleteRoleAssignmentRequest{
			AssignmentID:   fixtureB.assignment.ID,
			OrganizationID: tenantA.Organization.ID,
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err))

		count, err := db.NewSelect().
			Model((*permission.UserRoleAssignment)(nil)).
			Where("id = ?", fixtureB.assignment.ID).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("a permission cannot be deleted through another tenant's role", func(t *testing.T) {
		err := repo.DeleteResourcePermission(ctx, repositories.DeleteResourcePermissionRequest{
			PermissionID: fixtureB.permission.ID,
			RoleID:       fixtureA.role.ID,
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err))

		count, err := db.NewSelect().
			Model((*permission.ResourcePermission)(nil)).
			Where("id = ?", fixtureB.permission.ID).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("a permission cannot be rewritten through another tenant's role", func(t *testing.T) {
		err := repo.UpdateResourcePermission(ctx, &permission.ResourcePermission{
			ID:         fixtureB.permission.ID,
			RoleID:     fixtureA.role.ID,
			Resource:   permission.ResourceShipment.String(),
			Operations: []permission.Operation{permission.OpRead, permission.OpDelete},
			DataScope:  permission.DataScopeOrganization,
			UpdatedAt:  timeutils.NowUnix(),
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err))

		stored := new(permission.ResourcePermission)
		require.NoError(t, db.NewSelect().
			Model(stored).
			Where("id = ?", fixtureB.permission.ID).
			Scan(ctx))
		assert.Equal(t, fixtureB.role.ID, stored.RoleID)
		assert.Equal(t, []permission.Operation{permission.OpRead}, stored.Operations)
	})

	t.Run("the owning organization can still delete its own rows", func(t *testing.T) {
		deleted, err := repo.DeleteAssignment(ctx, repositories.DeleteRoleAssignmentRequest{
			AssignmentID:   fixtureA.assignment.ID,
			OrganizationID: tenantA.Organization.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, tenantA.User.ID, deleted.UserID)

		require.NoError(t, repo.DeleteResourcePermission(ctx,
			repositories.DeleteResourcePermissionRequest{
				PermissionID: fixtureA.permission.ID,
				RoleID:       fixtureA.role.ID,
			}))
	})
}
