//go:build integration

package shipmentmeter_test

import (
	"math"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmeter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type organizationRow struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

func TestShipmentsAreCountedForTheInstanceAndForOneOrganization(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	system := dbscope.WithSystem(ctx, "test")
	o := tenantdb.Organizations
	org, err := keel.IntoOne[organizationRow](
		system,
		db,
		o.AllTenants().Pick(o.ID, o.BusinessUnitID).Limit(1),
	)
	require.NoError(t, err)
	tenant := dbscope.Tenant{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID}

	s := shipmentdb.Shipments
	all, err := s.AllTenants().Count(system, db)
	require.NoError(t, err)
	own, err := s.Query().Count(dbscope.WithTenant(ctx, tenant), db)
	require.NoError(t, err)
	require.Positive(t, own, "the development fixtures have shipments")

	counter := shipmentmeter.New(shipmentmeter.Params{DB: postgres.WrapDB(db)}).Counter
	countFor := func(scope managedscope.Scope) int64 {
		t.Helper()
		counts, countErr := counter.CountByHour(ctx, &managedusageports.HourCountRequest{
			Scope: scope,
			Since: 0,
			Until: math.MaxInt64,
		})
		require.NoError(t, countErr)
		var sum int64
		for _, c := range counts {
			sum += c.Quantity
		}
		return sum
	}

	assert.Equal(
		t,
		all,
		countFor(managedscope.Instance()),
		"the instance counts every organization",
	)
	assert.Equal(
		t,
		own,
		countFor(managedscope.Organization(tenant)),
		"an organization counts its own",
	)
	assert.Zero(t, countFor(managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})), "and nothing of another's")
}
