//go:build integration

package organizationmeter_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/organizationmeter"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type organizationRow struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

func TestOrganizationsAreCountedForTheInstanceAndForOneBusinessUnit(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	first := seedtest.SeedFullTestData(t, ctx, db)
	seedtest.SeedAdditionalTenant(t, ctx, db, "OM")

	system := dbscope.WithSystem(ctx, "test")
	o := tenantdb.Organizations
	all, err := o.AllTenants().Count(system, db)
	require.NoError(t, err)
	org := organizationRow{ID: first.Organization.ID, BusinessUnitID: first.BusinessUnit.ID}
	inUnit, err := o.AllTenants().Where(o.BusinessUnitID.Eq(org.BusinessUnitID)).Count(system, db)
	require.NoError(t, err)

	reader := organizationmeter.New(organizationmeter.Params{DB: postgres.WrapDB(db)}).Reader
	levelFor := func(scope managedscope.Scope) int64 {
		t.Helper()
		level, levelErr := reader.Level(ctx, scope)
		require.NoError(t, levelErr)
		return level
	}

	assert.Equal(t, all, levelFor(managedscope.Instance()))
	assert.Equal(t, inUnit, levelFor(managedscope.Organization(dbscope.Tenant{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
	})), "an organization's scope counts the organizations of its business unit")
	assert.Less(t, inUnit, all, "and not the other business unit's")
	assert.Zero(t, levelFor(managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})))
}
