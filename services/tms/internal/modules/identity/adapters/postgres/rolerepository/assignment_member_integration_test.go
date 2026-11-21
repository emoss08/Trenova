//go:build integration

package rolerepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRoleRepository_AssignsOnlyLiveMembersOfTheOrganization(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "RM")
	seedtest.Join(t, ctx, db, tenantA.User)
	seedtest.Join(t, ctx, db, tenantB.User)
	outsider := seedtest.NewUser(tenantA.Organization.ID, tenantA.BusinessUnit.ID).
		WithUsername("not-a-member").
		WithEmail("not-a-member@example.com").
		Build(t, ctx, db)
	lapsed := seedtest.NewUser(tenantA.Organization.ID, tenantA.BusinessUnit.ID).
		WithUsername("lapsed-member").
		WithEmail("lapsed-member@example.com").
		Build(t, ctx, db)
	ended := time.Now().Add(-time.Hour).Unix()
	require.NoError(t, tenantdb.OrganizationMemberships.Create(systemCtx(ctx), db,
		&tenant.OrganizationMembership{
			BusinessUnitID: lapsed.BusinessUnitID,
			UserID:         lapsed.ID,
			OrganizationID: tenantA.Organization.ID,
			ExpiresAt:      &ended,
		}))
	sa := tenantdb.SystemActors
	actor, err := sa.AllTenants().Pick(sa.UserID).
		Where(sa.BusinessUnitID.Eq(tenantA.BusinessUnit.ID)).One(systemCtx(ctx), db)
	require.NoError(t, err)

	repo := New(Params{
		Entitlements: entitlementtest.Instance(),
		DB:           postgres.WrapDB(db),
		Logger:       zap.NewNop(),
	})
	tctx := tenantA.Ctx(ctx)
	role := newTestRole("Assignable", nil, "shipment")
	require.NoError(t, repo.Create(tctx, role))
	assign := func(userID pulid.ID) error {
		return repo.CreateAssignment(tctx, &rbac.UserRoleAssignment{
			UserID:         userID,
			OrganizationID: tenantA.Organization.ID,
			RoleID:         role.ID,
			AssignedBy:     tenantA.User.ID,
		})
	}

	unknown := assign(pulid.MustNew("usr_"))
	require.Error(t, unknown)
	assertSameRefusal := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		assert.Equal(t, unknown.Error(), err.Error(), "it answers exactly as for an unknown person")
		var got, want *errortypes.Error
		require.ErrorAs(t, err, &got)
		require.ErrorAs(t, unknown, &want)
		assert.Equal(t, want.Field(), got.Field())
		assert.Equal(t, want.Code, got.Code)
	}

	t.Run("a person of another tenant is refused", func(t *testing.T) {
		assertSameRefusal(t, assign(tenantB.User.ID))
	})
	t.Run("a person of the business unit who is not a member is refused", func(t *testing.T) {
		assertSameRefusal(t, assign(outsider.ID))
	})
	t.Run("a member whose membership ended is refused", func(t *testing.T) {
		assertSameRefusal(t, assign(lapsed.ID))
	})
	t.Run("the tenant's system actor is refused", func(t *testing.T) {
		assertSameRefusal(t, assign(actor.UserID))
	})
	t.Run("a live member is assigned and the row carries the business unit", func(t *testing.T) {
		require.NoError(t, assign(tenantA.User.ID))
		ura := rbacdb.UserRoleAssignments
		held, err := ura.AllTenants().Where(
			ura.UserID.Eq(tenantA.User.ID),
			ura.RoleID.Eq(role.ID),
		).One(systemCtx(ctx), db)
		require.NoError(t, err)
		assert.Equal(t, tenantA.BusinessUnit.ID, held.BusinessUnitID)
	})
	t.Run("nothing was written for anyone refused", func(t *testing.T) {
		ura := rbacdb.UserRoleAssignments
		count, err := ura.AllTenants().Where(
			ura.RoleID.Eq(role.ID),
			ura.UserID.In(tenantB.User.ID, outsider.ID, lapsed.ID, actor.UserID),
		).Count(systemCtx(ctx), db)
		require.NoError(t, err)
		assert.Zero(t, count)
	})
}

func TestTheDatabaseRefusesAnAssignmentOfAnotherTenantsPerson(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "RD")
	role := newTestRole("Held by the key", nil, "shipment")
	require.NoError(t, rbacdb.Roles.Create(tenantA.Ctx(ctx), db, role))

	err := rbacdb.UserRoleAssignments.Create(systemCtx(ctx), db, &rbac.UserRoleAssignment{
		UserID:         tenantB.User.ID,
		OrganizationID: tenantA.Organization.ID,
		BusinessUnitID: tenantA.BusinessUnit.ID,
		RoleID:         role.ID,
	})
	require.Error(t, err, "the person key carries the business unit")

	err = rbacdb.UserRoleAssignments.Create(systemCtx(ctx), db, &rbac.UserRoleAssignment{
		UserID:         tenantB.User.ID,
		OrganizationID: tenantA.Organization.ID,
		BusinessUnitID: tenantB.BusinessUnit.ID,
		RoleID:         role.ID,
	})
	require.Error(t, err, "the organization key holds the business unit to the organization")
}
