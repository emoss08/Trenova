//go:build integration

package recordvisibilityrepository

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sources = kernelports.VisibleRecordSources{
	{Kind: "shipment", ID: shipmentdb.Shipments.ID},
	{Kind: "worker", ID: workerdb.Workers.ID},
}

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
	sh := shipmentdb.Shipments
	_, err := sh.Where(sh.ID.Eq(owned.ID)).
		Update(data.Ctx(ctx), admin, sh.OwnerID.To(data.User.ID))
	require.NoError(t, err)

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn), Sources: sources})
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

// A person who cannot read shipments or workers is sent no read scope for
// them, so the database shows every row of both kinds: a record that links to
// one shows its reference view. Whether the person may see such a record as a
// search hit is the caller's decision (globalsearchservice refuses a kind the
// person cannot read before it asks), not the database's.
func TestVisibleIDs_AKindThePersonCannotReadIsLeftToTheCaller(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	newWorker := func(first string, userID pulid.ID) *worker.Worker {
		entity := &worker.Worker{
			ID:             pulid.MustNew("wrk_"),
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			StateID:        data.State.ID,
			UserID:         userID,
			Status:         domaintypes.StatusActive,
			Type:           worker.WorkerTypeEmployee,
			DriverType:     worker.DriverTypeOTR,
			FirstName:      first,
			LastName:       "Driver",
			AddressLine1:   "1 Depot Road",
			City:           "Springfield",
			PostalCode:     "62701",
			Email:          strings.ToLower(first) + "-" + pulid.MustNew("e").String() + "@example.com",
			PhoneNumber:    "555-0100",
			Gender:         worker.GenderMale,
			CanBeAssigned:  true,
		}
		require.NoError(t, workerdb.Workers.Create(data.Ctx(ctx), admin, entity))
		return entity
	}
	me := newWorker("Mine", data.User.ID)
	colleague := newWorker("Other", pulid.Nil)

	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, admin, data, tenant)
	newShipment := func(code string) *testutil.ShipmentGraph {
		return testutil.CreateShipmentGraph(t, ctx, admin, fixture, tenant, testutil.ShipmentGraphParams{
			BOL:          "BOL-" + code,
			ProNumber:    "PRO-" + code,
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusAssigned},
		})
	}
	driven := newShipment("DRIVEN")
	unassigned := newShipment("OTHER")
	err := shipmentdb.Assignments.Create(data.Ctx(ctx), admin, &shipment.Assignment{
		ID:              pulid.MustNew("a_"),
		BusinessUnitID:  tenant.BuID,
		OrganizationID:  tenant.OrgID,
		ShipmentMoveID:  driven.Moves[0].ID,
		PrimaryWorkerID: &me.ID,
		Status:          shipment.AssignmentStatusNew,
	})
	require.NoError(t, err)

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn), Sources: sources})
	cannotRead := readscope.With(
		dbscope.WithTenant(t.Context(), tenant.DBTenant()),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{}, nil
		},
	)

	shipments, err := repo.VisibleIDs(cannotRead, &kernelports.ListVisibleRecordsRequest{
		TenantInfo: tenant,
		Kind:       "shipment",
		IDs:        []pulid.ID{driven.Shipment.ID, unassigned.Shipment.ID},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{driven.Shipment.ID, unassigned.Shipment.ID}, shipments)

	workers, err := repo.VisibleIDs(cannotRead, &kernelports.ListVisibleRecordsRequest{
		TenantInfo: tenant,
		Kind:       "worker",
		IDs:        []pulid.ID{me.ID, colleague.ID},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{me.ID, colleague.ID}, workers)

	ownShipments := readscope.With(
		dbscope.WithTenant(t.Context(), tenant.DBTenant()),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{"shipment": {Scope: readscope.ScopeOwn}}, nil
		},
	)
	owned, err := repo.VisibleIDs(ownShipments, &kernelports.ListVisibleRecordsRequest{
		TenantInfo: tenant,
		Kind:       "shipment",
		IDs:        []pulid.ID{driven.Shipment.ID, unassigned.Shipment.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, owned, "an own scope still narrows: the person owns neither shipment")
}
