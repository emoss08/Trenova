//go:build integration

package recordvisibilityrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisibleIDs_AppliesTheRequestsReadScope(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, admin, data, tenant)
	owned := testutil.CreateShipmentGraph(t, ctx, admin, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:        "BOL-OWNED",
		ProNumber:  "PRO-OWNED",
		ShipmentID: pulid.MustNew("shp_"),
	}).Shipment
	others := testutil.CreateShipmentGraph(t, ctx, admin, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:        "BOL-OTHER",
		ProNumber:  "PRO-OTHER",
		ShipmentID: pulid.MustNew("shp_"),
	}).Shipment
	_, err := admin.NewUpdate().
		Table(shipmentcol.ShipmentTable.Name).
		Set(shipmentcol.ShipmentColumns.OwnerID.Set(), data.User.ID).
		Where("id = ?", owned.ID).
		Exec(ctx)
	require.NoError(t, err)

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn)})
	ids := []pulid.ID{owned.ID, others.ID, pulid.MustNew("shp_")}
	request := &kernelports.ListVisibleRecordsRequest{TenantInfo: tenant, Kind: "shipment", IDs: ids}

	ownScope := readscope.With(
		dbscope.WithTenant(t.Context(), tenant.DBTenant()),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{"shipment": {Scope: readscope.ScopeOwn}}, nil
		},
	)
	visible, err := repo.VisibleIDs(ownScope, request)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{owned.ID}, visible)

	visible, err = repo.VisibleIDs(dbscope.WithTenant(t.Context(), tenant.DBTenant()), request)
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{owned.ID, others.ID}, visible)

	otherTenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: tenant.BuID}
	visible, err = repo.VisibleIDs(
		dbscope.WithTenant(t.Context(), otherTenant.DBTenant()),
		&kernelports.ListVisibleRecordsRequest{TenantInfo: otherTenant, Kind: "shipment", IDs: ids},
	)
	require.NoError(t, err)
	assert.Empty(t, visible)
}
