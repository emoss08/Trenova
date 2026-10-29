//go:build integration

package carriersettlementrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicarrierinvoicerepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice/edicarrierinvoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/carrierinvoicematchrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/carriersettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type holdFixture struct {
	ctx       context.Context
	db        *keel.DB
	tenant    pagination.TenantInfo
	carrierID pulid.ID
	moveIDs   []pulid.ID
	partnerID pulid.ID
	costs     carriersettlementports.CarrierCostEventRepository
	matches   carriersettlementports.CarrierInvoiceMatchRepository
	invoices  ediports.EDICarrierInvoiceRepository
}

const holdMovesSQL = "SELECT sm.id FROM shipment_moves AS sm " +
	"WHERE sm.organization_id = $1 AND sm.business_unit_id = $2 " +
	"AND NOT EXISTS (SELECT 1 FROM carrier_assignments AS ca " +
	"WHERE ca.shipment_move_id = sm.id AND ca.status <> 'Canceled') LIMIT 4"

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func setupHoldFixture(t *testing.T) *holdFixture {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	rows, err := keel.QueryAll(
		ctx,
		db,
		pgx.RowToStructByPos[tenantRow],
		"SELECT organization_id, business_unit_id FROM shipment_moves "+
			"ORDER BY organization_id, business_unit_id LIMIT 1",
	)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	tenant := pagination.TenantInfo{OrgID: rows[0].OrganizationID, BuID: rows[0].BusinessUnitID}
	tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())

	moveIDs, err := keel.QueryAll(
		ctx,
		db,
		pgx.RowTo[pulid.ID],
		holdMovesSQL,
		tenant.OrgID.String(),
		tenant.BuID.String(),
	)
	require.NoError(t, err)
	require.Len(t, moveIDs, 4, "the development seed has moves without a carrier")

	carrierID, err := keel.QueryValue[pulid.ID](
		ctx,
		db,
		"SELECT id FROM carriers WHERE organization_id = $1 AND business_unit_id = $2 LIMIT 1",
		tenant.OrgID.String(),
		tenant.BuID.String(),
	)
	require.NoError(t, err)

	partner := &edi.EDIPartner{
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		Kind:           edi.PartnerKindExternal,
		Code:           "HOLD-TEST",
		Name:           "Hold Test Partner",
	}
	require.NoError(t, edidb.EDIPartners.Create(tenantCtx, db, partner))

	conn := postgres.NewTestConnection(db)
	return &holdFixture{
		ctx:       tenantCtx,
		db:        db,
		tenant:    tenant,
		carrierID: carrierID,
		moveIDs:   moveIDs,
		partnerID: partner.ID,
		costs: carriersettlementrepository.NewCostEvent(carriersettlementrepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
		matches: carrierinvoicematchrepository.New(carrierinvoicematchrepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
		invoices: edicarrierinvoicerepository.New(edicarrierinvoicerepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
	}
}

func (f *holdFixture) assignment(t *testing.T) pulid.ID {
	t.Helper()
	require.NotEmpty(t, f.moveIDs, "fixture ran out of moves")
	moveID := f.moveIDs[0]
	f.moveIDs = f.moveIDs[1:]
	assignment := &shipment.CarrierAssignment{
		ID:             pulid.MustNew("ca_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		ShipmentMoveID: moveID,
		CarrierID:      f.carrierID,
		Status:         shipmentvocab.CarrierAssignmentStatusConfirmed,
		RateMethod:     shipmentvocab.CarrierRateMethodFlat,
		BaseRate:       decimal.NewFromInt(1000),
		TotalCost:      decimal.NewFromInt(1000),
		CurrencyCode:   "USD",
	}
	require.NoError(t, shipmentdb.CarrierAssignments.Create(f.ctx, f.db, assignment))
	return assignment.ID
}

func (f *holdFixture) costEvent(
	t *testing.T,
	assignmentID *pulid.ID,
	settlementID *pulid.ID,
) pulid.ID {
	t.Helper()
	event := &carriersettlement.CostEvent{
		ID:                  pulid.MustNew("cce_"),
		OrganizationID:      f.tenant.OrgID,
		BusinessUnitID:      f.tenant.BuID,
		CarrierID:           f.carrierID,
		CarrierAssignmentID: assignmentID,
		SettlementID:        settlementID,
		EventType:           carriersettlement.CostEventTypeAdjustment,
		Status:              carriersettlement.CostEventStatusPending,
		IdempotencyKey:      pulid.MustNew("idk_").String(),
		EventDate:           1_700_000_000,
		AmountMinor:         100_000,
		CurrencyCode:        "USD",
	}
	if settlementID != nil {
		event.Status = carriersettlement.CostEventStatusAttached
	}
	require.NoError(t, carriersettlementdb.CostEvents.Create(f.ctx, f.db, event))
	return event.ID
}

func (f *holdFixture) match(
	t *testing.T,
	assignmentID pulid.ID,
	invoiceNumber string,
	status carriersettlement.InvoiceMatchStatus,
) (*carriersettlement.InvoiceMatch, error) {
	t.Helper()
	extractionID := pulid.MustNew("dax_")
	return f.matches.Create(f.ctx, &carriersettlement.InvoiceMatch{
		OrganizationID:         f.tenant.OrgID,
		BusinessUnitID:         f.tenant.BuID,
		DocumentAIExtractionID: &extractionID,
		CarrierID:              f.carrierID,
		CarrierAssignmentID:    assignmentID,
		Status:                 status,
		MatchedVia:             carriersettlement.MatchViaManual,
		InvoiceNumber:          invoiceNumber,
		CurrencyCode:           "USD",
	})
}

func eventIDs(events []*carriersettlement.CostEvent) []pulid.ID {
	ids := make([]pulid.ID, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}

func TestHoldUntilInvoiceMatched_Integration(t *testing.T) {
	f := setupHoldFixture(t)

	resolvedLoad := f.assignment(t)
	openLoad := f.assignment(t)
	bareLoad := f.assignment(t)
	_, err := f.match(t, resolvedLoad, "INV-R", carriersettlement.InvoiceMatchStatusResolved)
	require.NoError(t, err)
	_, err = f.match(t, openLoad, "INV-O", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)

	resolvedEvent := f.costEvent(t, &resolvedLoad, nil)
	openEvent := f.costEvent(t, &openLoad, nil)
	bareEvent := f.costEvent(t, &bareLoad, nil)
	manualEvent := f.costEvent(t, nil, nil)

	held, err := f.costs.ListPendingByCarrier(
		f.ctx,
		&carriersettlementports.ListPendingCostEventsRequest{
			TenantInfo:              f.tenant,
			CarrierID:               f.carrierID,
			PeriodEnd:               1_800_000_000,
			HoldUntilInvoiceMatched: true,
		},
	)
	require.NoError(t, err)
	assert.Subset(t, eventIDs(held), []pulid.ID{resolvedEvent, manualEvent})
	assert.NotContains(t, eventIDs(held), openEvent)
	assert.NotContains(t, eventIDs(held), bareEvent)

	all, err := f.costs.ListPendingByCarrier(
		f.ctx,
		&carriersettlementports.ListPendingCostEventsRequest{
			TenantInfo: f.tenant,
			CarrierID:  f.carrierID,
			PeriodEnd:  1_800_000_000,
		},
	)
	require.NoError(t, err)
	assert.Subset(t, eventIDs(all), []pulid.ID{resolvedEvent, openEvent, bareEvent, manualEvent})

	carriers, err := f.costs.ListCarrierIDsWithPendingEvents(
		f.ctx,
		carriersettlementports.ListCarriersWithPendingEventsRequest{
			TenantInfo:              f.tenant,
			PeriodEnd:               1_800_000_000,
			HoldUntilInvoiceMatched: true,
		},
	)
	require.NoError(t, err)
	assert.Contains(t, carriers, f.carrierID)

	settlementID := pulid.MustNew("cs_")
	f.costEvent(t, &resolvedLoad, &settlementID)
	f.costEvent(t, &openLoad, &settlementID)
	f.costEvent(t, &openLoad, &settlementID)
	f.costEvent(t, nil, &settlementID)
	awaiting, err := f.costs.CountAwaitingInvoiceMatch(f.ctx, f.tenant, settlementID)
	require.NoError(t, err)
	assert.Equal(t, 1, awaiting, "one load, counted once, still waits on its match")

	resolved, err := f.matches.ListResolvedAssignmentIDs(
		f.ctx,
		carriersettlementports.ListResolvedMatchAssignmentsRequest{
			TenantInfo:    f.tenant,
			AssignmentIDs: []pulid.ID{resolvedLoad, openLoad, bareLoad},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{resolvedLoad}, resolved)
}

func TestCarrierInvoiceDuplicates_Integration(t *testing.T) {
	f := setupHoldFixture(t)
	load := f.assignment(t)
	otherLoad := f.assignment(t)

	first, err := f.match(t, load, "inv 1001", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)

	live, err := f.matches.GetLiveByCarrierInvoiceNumber(
		f.ctx,
		&carriersettlementports.GetLiveCarrierInvoiceMatchByNumberRequest{
			TenantInfo:    f.tenant,
			CarrierID:     f.carrierID,
			InvoiceNumber: " INV  1001 ",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, live)
	assert.Equal(t, first.ID, live.ID)

	_, err = f.match(t, otherLoad, "INV 1001", carriersettlement.InvoiceMatchStatusMatched)
	require.Error(t, err)
	assert.True(
		t,
		dberror.IsUniqueConstraintViolation(err),
		"a carrier's invoice number holds one live match",
	)

	_, err = f.match(t, otherLoad, "INV 1002", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)
	onLoad, err := f.matches.ListLiveByAssignment(
		f.ctx,
		carriersettlementports.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
			TenantInfo:   f.tenant,
			AssignmentID: load,
		},
	)
	require.NoError(t, err)
	require.Len(t, onLoad, 1)

	first.Status = carriersettlement.InvoiceMatchStatusRejected
	_, err = f.matches.Update(f.ctx, first)
	require.NoError(t, err)
	_, err = f.match(t, otherLoad, "INV 1001", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err, "a rejected match frees its number")

	invoice := func(number string) *edicarrierinvoice.CarrierInvoice {
		return &edicarrierinvoice.CarrierInvoice{
			OrganizationID:   f.tenant.OrgID,
			BusinessUnitID:   f.tenant.BuID,
			EDIPartnerID:     f.partnerID,
			InboundMessageID: pulid.MustNew("edim_"),
			InvoiceNumber:    number,
		}
	}
	recorded, err := f.invoices.CreateCarrierInvoice(f.ctx, invoice("A-77"))
	require.NoError(t, err)
	_, err = f.invoices.CreateCarrierInvoice(f.ctx, invoice(" a-77"))
	require.Error(t, err)
	assert.True(
		t,
		dberror.IsUniqueConstraintViolation(err),
		"a partner's invoice number is recorded once",
	)

	found, err := f.invoices.GetCarrierInvoiceByNumber(
		f.ctx,
		&ediports.GetEDICarrierInvoiceByNumberRequest{
			TenantInfo:    f.tenant,
			PartnerID:     f.partnerID,
			InvoiceNumber: "a-77 ",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, recorded.ID, found.ID)
}

func TestCarrierInvoiceDuplicatesMigration_MarksExistingRepeats_Integration(t *testing.T) {
	f := setupHoldFixture(t)
	load := f.assignment(t)
	otherLoad := f.assignment(t)

	for _, statement := range []string{
		`DROP INDEX IF EXISTS "uq_edi_carrier_invoices_partner_invoice_number"`,
		`DROP INDEX IF EXISTS "uq_carrier_invoice_matches_carrier_invoice_number"`,
	} {
		_, err := keel.Exec(f.ctx, f.db, statement)
		require.NoError(t, err)
	}

	invoiceIDs := make([]pulid.ID, 0, 2)
	for i, number := range []string{"R-9", " r-9"} {
		invoice := &edicarrierinvoice.CarrierInvoice{
			ID:               pulid.MustNew("edici_"),
			OrganizationID:   f.tenant.OrgID,
			BusinessUnitID:   f.tenant.BuID,
			EDIPartnerID:     f.partnerID,
			InboundMessageID: pulid.MustNew("edim_"),
			InvoiceNumber:    number,
		}
		require.NoError(t, edicarrierinvoicedb.CarrierInvoices.Create(f.ctx, f.db, invoice))
		_, err := keel.Exec(
			f.ctx,
			f.db,
			"UPDATE edi_carrier_invoices SET created_at = $1 WHERE id = $2",
			int64(1_700_000_000+i),
			invoice.ID.String(),
		)
		require.NoError(t, err)
		invoiceIDs = append(invoiceIDs, invoice.ID)
	}
	firstMatch, err := f.match(t, load, "M-5", carriersettlement.InvoiceMatchStatusResolved)
	require.NoError(t, err)
	repeatMatch, err := f.match(t, otherLoad, "m-5", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)

	migration, err := migrations.File("20261231007190_carrier_invoice_duplicates")
	require.NoError(t, err)
	_, err = keel.Exec(f.ctx, f.db, migration)
	require.NoError(t, err)

	duplicateOf, err := keel.QueryValue[pulid.ID](
		f.ctx,
		f.db,
		"SELECT duplicate_of_id FROM edi_carrier_invoices WHERE id = $1",
		invoiceIDs[1].String(),
	)
	require.NoError(t, err)
	assert.Equal(t, invoiceIDs[0], duplicateOf, "the later copy points at the first")

	repeat, err := f.matches.GetByID(
		f.ctx,
		carriersettlementports.GetCarrierInvoiceMatchByIDRequest{
			ID:         repeatMatch.ID,
			TenantInfo: f.tenant,
		},
	)
	require.NoError(t, err)
	require.True(t, repeat.IsDuplicate())
	assert.Equal(t, firstMatch.ID, *repeat.DuplicateOfMatchID)
}
