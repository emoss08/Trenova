//go:build integration

package carriersettlementrepository_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/carrierinvoicematchrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/carriersettlementrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/edicarrierinvoicerepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type holdFixture struct {
	ctx       context.Context
	db        *bun.DB
	tenant    pagination.TenantInfo
	carrierID pulid.ID
	moveIDs   []pulid.ID
	partnerID pulid.ID
	costs     repositories.CarrierCostEventRepository
	matches   repositories.CarrierInvoiceMatchRepository
	invoices  repositories.EDICarrierInvoiceRepository
}

func setupHoldFixture(t *testing.T) *holdFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "test-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var tenantRow struct {
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("shipment_moves").
		ColumnExpr("organization_id, business_unit_id").
		Limit(1).
		Scan(ctx, &tenantRow))
	tenant := pagination.TenantInfo{OrgID: tenantRow.OrganizationID, BuID: tenantRow.BusinessUnitID}

	var moveIDs []pulid.ID
	require.NoError(t, db.NewSelect().
		TableExpr("shipment_moves AS sm").
		Column("sm.id").
		Where("sm.organization_id = ?", tenant.OrgID).
		Where("sm.business_unit_id = ?", tenant.BuID).
		Where("NOT EXISTS (SELECT 1 FROM carrier_assignments AS ca WHERE ca.shipment_move_id = sm.id AND ca.status <> 'Canceled')").
		Limit(4).
		Scan(ctx, &moveIDs))
	require.Len(t, moveIDs, 4, "the development seed has moves without a carrier")

	var carrierID pulid.ID
	require.NoError(t, db.NewSelect().
		TableExpr("carriers").
		Column("id").
		Where("organization_id = ?", tenant.OrgID).
		Where("business_unit_id = ?", tenant.BuID).
		Limit(1).
		Scan(ctx, &carrierID))

	partner := &edi.EDIPartner{
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		Kind:           edi.PartnerKindExternal,
		Code:           "HOLD-TEST",
		Name:           "Hold Test Partner",
	}
	_, err = db.NewInsert().Model(partner).Exec(ctx)
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	return &holdFixture{
		ctx:       ctx,
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
		Status:         shipment.CarrierAssignmentStatusConfirmed,
		RateMethod:     shipment.CarrierRateMethodFlat,
		BaseRate:       decimal.NewFromInt(1000),
		TotalCost:      decimal.NewFromInt(1000),
		CurrencyCode:   "USD",
	}
	_, err := f.db.NewInsert().Model(assignment).Exec(f.ctx)
	require.NoError(t, err)
	return assignment.ID
}

func (f *holdFixture) costEvent(t *testing.T, assignmentID *pulid.ID, settlementID *pulid.ID) pulid.ID {
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
	_, err := f.db.NewInsert().Model(event).Exec(f.ctx)
	require.NoError(t, err)
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

	held, err := f.costs.ListPendingByCarrier(f.ctx, &repositories.ListPendingCostEventsRequest{
		TenantInfo:              f.tenant,
		CarrierID:               f.carrierID,
		PeriodEnd:               1_800_000_000,
		HoldUntilInvoiceMatched: true,
	})
	require.NoError(t, err)
	assert.Subset(t, eventIDs(held), []pulid.ID{resolvedEvent, manualEvent})
	assert.NotContains(t, eventIDs(held), openEvent)
	assert.NotContains(t, eventIDs(held), bareEvent)

	all, err := f.costs.ListPendingByCarrier(f.ctx, &repositories.ListPendingCostEventsRequest{
		TenantInfo: f.tenant,
		CarrierID:  f.carrierID,
		PeriodEnd:  1_800_000_000,
	})
	require.NoError(t, err)
	assert.Subset(t, eventIDs(all), []pulid.ID{resolvedEvent, openEvent, bareEvent, manualEvent})

	carriers, err := f.costs.ListCarrierIDsWithPendingEvents(f.ctx, repositories.ListCarriersWithPendingEventsRequest{
		TenantInfo:              f.tenant,
		PeriodEnd:               1_800_000_000,
		HoldUntilInvoiceMatched: true,
	})
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

	resolved, err := f.matches.ListResolvedAssignmentIDs(f.ctx, repositories.ListResolvedMatchAssignmentsRequest{
		TenantInfo:    f.tenant,
		AssignmentIDs: []pulid.ID{resolvedLoad, openLoad, bareLoad},
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{resolvedLoad}, resolved)
}

func TestCarrierInvoiceDuplicates_Integration(t *testing.T) {
	f := setupHoldFixture(t)
	load := f.assignment(t)
	otherLoad := f.assignment(t)

	first, err := f.match(t, load, "inv 1001", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)

	live, err := f.matches.GetLiveByCarrierInvoiceNumber(f.ctx, &repositories.GetLiveCarrierInvoiceMatchByNumberRequest{
		TenantInfo:    f.tenant,
		CarrierID:     f.carrierID,
		InvoiceNumber: " INV  1001 ",
	})
	require.NoError(t, err)
	require.NotNil(t, live)
	assert.Equal(t, first.ID, live.ID)

	_, err = f.match(t, otherLoad, "INV 1001", carriersettlement.InvoiceMatchStatusMatched)
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err), "a carrier's invoice number holds one live match")

	_, err = f.match(t, otherLoad, "INV 1002", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)
	onLoad, err := f.matches.ListLiveByAssignment(f.ctx, repositories.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
		TenantInfo:   f.tenant,
		AssignmentID: load,
	})
	require.NoError(t, err)
	require.Len(t, onLoad, 1)

	first.Status = carriersettlement.InvoiceMatchStatusRejected
	_, err = f.matches.Update(f.ctx, first)
	require.NoError(t, err)
	_, err = f.match(t, otherLoad, "INV 1001", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err, "a rejected match frees its number")

	invoice := func(number string) *edi.CarrierInvoice {
		return &edi.CarrierInvoice{
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
	assert.True(t, dberror.IsUniqueConstraintViolation(err), "a partner's invoice number is recorded once")

	found, err := f.invoices.GetCarrierInvoiceByNumber(f.ctx, &repositories.GetEDICarrierInvoiceByNumberRequest{
		TenantInfo:    f.tenant,
		PartnerID:     f.partnerID,
		InvoiceNumber: "a-77 ",
	})
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
		_, err := f.db.ExecContext(f.ctx, statement)
		require.NoError(t, err)
	}

	invoiceIDs := make([]pulid.ID, 0, 2)
	for i, number := range []string{"R-9", " r-9"} {
		invoice := &edi.CarrierInvoice{
			ID:               pulid.MustNew("edici_"),
			OrganizationID:   f.tenant.OrgID,
			BusinessUnitID:   f.tenant.BuID,
			EDIPartnerID:     f.partnerID,
			InboundMessageID: pulid.MustNew("edim_"),
			InvoiceNumber:    number,
			CreatedAt:        int64(1_700_000_000 + i),
		}
		_, err := f.db.NewInsert().Model(invoice).Exec(f.ctx)
		require.NoError(t, err)
		invoiceIDs = append(invoiceIDs, invoice.ID)
	}
	firstMatch, err := f.match(t, load, "M-5", carriersettlement.InvoiceMatchStatusResolved)
	require.NoError(t, err)
	repeatMatch, err := f.match(t, otherLoad, "m-5", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)

	_, thisFile, _, _ := runtime.Caller(0)
	migration, err := os.ReadFile(filepath.Join(
		filepath.Dir(thisFile),
		"../../migrations/20261231007190_carrier_invoice_duplicates.tx.up.sql",
	))
	require.NoError(t, err)
	for _, statement := range strings.Split(string(migration), "--bun:split") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		_, err = f.db.ExecContext(f.ctx, statement)
		require.NoError(t, err, statement)
	}

	var duplicateOf pulid.ID
	require.NoError(t, f.db.NewSelect().
		TableExpr("edi_carrier_invoices").
		Column("duplicate_of_id").
		Where("id = ?", invoiceIDs[1]).
		Scan(f.ctx, &duplicateOf))
	assert.Equal(t, invoiceIDs[0], duplicateOf, "the later copy points at the first")

	repeat, err := f.matches.GetByID(f.ctx, repositories.GetCarrierInvoiceMatchByIDRequest{
		ID:         repeatMatch.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	require.True(t, repeat.IsDuplicate())
	assert.Equal(t, firstMatch.ID, *repeat.DuplicateOfMatchID)
}
