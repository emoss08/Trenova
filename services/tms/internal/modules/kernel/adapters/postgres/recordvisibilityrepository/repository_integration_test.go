//go:build integration

package recordvisibilityrepository

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

func TestVisibleIDs_ANoneScopeLeavesOnlyWhatIsThePersonsOwn(t *testing.T) {
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
		_, err := admin.NewInsert().Model(entity).Exec(ctx)
		require.NoError(t, err)
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
	_, err := admin.NewInsert().Model(&shipment.Assignment{
		ID:              pulid.MustNew("a_"),
		BusinessUnitID:  tenant.BuID,
		OrganizationID:  tenant.OrgID,
		ShipmentMoveID:  driven.Moves[0].ID,
		PrimaryWorkerID: &me.ID,
		Status:          shipment.AssignmentStatusNew,
	}).Exec(ctx)
	require.NoError(t, err)

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn)})
	cannotRead := readscope.With(
		dbscope.WithTenant(t.Context(), tenant.DBTenant()),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{
				"shipment": {Scope: readscope.ScopeNone},
				"worker":   {Scope: readscope.ScopeNone},
				"order":    {Scope: readscope.ScopeNone},
				"invoice":  {Scope: readscope.ScopeNone},
			}, nil
		},
	)

	shipments, err := repo.VisibleIDs(cannotRead, &kernelports.ListVisibleRecordsRequest{
		TenantInfo: tenant,
		Kind:       "shipment",
		IDs:        []pulid.ID{driven.Shipment.ID, unassigned.Shipment.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{driven.Shipment.ID}, shipments)

	workers, err := repo.VisibleIDs(cannotRead, &kernelports.ListVisibleRecordsRequest{
		TenantInfo: tenant,
		Kind:       "worker",
		IDs:        []pulid.ID{me.ID, colleague.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{me.ID}, workers)

	everything, err := repo.VisibleIDs(
		dbscope.WithTenant(t.Context(), tenant.DBTenant()),
		&kernelports.ListVisibleRecordsRequest{
			TenantInfo: tenant,
			Kind:       "worker",
			IDs:        []pulid.ID{me.ID, colleague.ID},
		},
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{me.ID, colleague.ID}, everything)
}
