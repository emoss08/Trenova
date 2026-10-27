//go:build integration

package recordsummaryrepository_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/recordsummaryrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaries_ReadEachRecordOnceWithItsLaneStatusAndMeasures(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	stp := shipmentdb.Stops
	first, err := stp.AllTenants().Order(stp.ID.Asc()).First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: first.OrganizationID, BuID: first.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	sp := shipmentdb.Shipments
	shipments, err := sp.Order(sp.ID.Asc()).Limit(3).List(ctx, db)
	require.NoError(t, err)
	require.NotEmpty(t, shipments, "the development seed should hold shipments")
	carriers, err := carrierdb.Carriers.Limit(2).List(ctx, db)
	require.NoError(t, err)
	entries, err := journalentrydb.JournalEntries.Limit(2).List(ctx, db)
	require.NoError(t, err)

	refs := map[permission.Resource][]pulid.ID{}
	for _, shipment := range shipments {
		refs[permission.ResourceShipment] = append(refs[permission.ResourceShipment], shipment.ID)
	}
	for _, carrier := range carriers {
		refs[permission.ResourceCarrier] = append(refs[permission.ResourceCarrier], carrier.ID)
	}
	for _, entry := range entries {
		refs[permission.ResourceJournalEntry] = append(refs[permission.ResourceJournalEntry], entry.ID)
	}
	missing := pulid.MustNew("shp_")
	refs[permission.ResourceShipment] = append(refs[permission.ResourceShipment], missing)

	repo := recordsummaryrepository.New(recordsummaryrepository.Params{
		DB: postgres.NewTestConnection(db),
	})
	summaries, err := repo.Summaries(ctx, &agentports.RecordSummaryRequest{
		Tenant: tenant,
		Refs:   refs,
		Now:    time.Now().Unix(),
	})
	require.NoError(t, err)

	read := summaries[permission.ResourceShipment]
	assert.NotContains(t, read, missing, "a record that does not exist is not read")
	for _, shipment := range shipments {
		summary, ok := read[shipment.ID]
		require.True(t, ok, "%s was not read", shipment.ProNumber)
		assert.Equal(t, shipment.ProNumber, summary.Label)
		assert.Equal(t, string(shipment.Status), summary.Status)
		if late, measured := summary.Measures[board.MeasureLateness]; measured {
			assert.Equal(t, board.ValueDuration, late.Value.Kind)
			assert.False(t, late.Value.Number.IsNegative())
		}
	}
	for _, carrier := range carriers {
		summary, ok := summaries[permission.ResourceCarrier][carrier.ID]
		require.True(t, ok)
		assert.Equal(t, carrier.Name, summary.Label)
		assert.Equal(t, string(carrier.ComplianceStatus), summary.Status)
	}
	for _, entry := range entries {
		summary, ok := summaries[permission.ResourceJournalEntry][entry.ID]
		require.True(t, ok)
		assert.Equal(t, entry.EntryNumber, summary.Label)
		amount := summary.Measures[board.MeasureAmount]
		assert.Equal(t, entry.TotalDebit, amount.Value.Number.Shift(2).IntPart())
	}
}

func TestSummaries_ReadNothingOfAnotherTenant(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sp := shipmentdb.Shipments
	shipment, err := sp.AllTenants().Order(sp.ID.Asc()).First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ctx = dbscope.WithTenant(ctx, other.DBTenant())
	repo := recordsummaryrepository.New(recordsummaryrepository.Params{
		DB: postgres.NewTestConnection(db),
	})

	summaries, err := repo.Summaries(ctx, &agentports.RecordSummaryRequest{
		Tenant: other,
		Refs:   map[permission.Resource][]pulid.ID{permission.ResourceShipment: {shipment.ID}},
		Now:    time.Now().Unix(),
	})

	require.NoError(t, err)
	assert.Empty(t, summaries[permission.ResourceShipment])
}
