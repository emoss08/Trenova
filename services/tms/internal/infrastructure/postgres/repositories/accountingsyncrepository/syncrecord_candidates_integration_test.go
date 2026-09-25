//go:build integration

package accountingsyncrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicerepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type candidateShipment struct {
	ID         pulid.ID `bun:"id"`
	CustomerID pulid.ID `bun:"customer_id"`
	ProNumber  string   `bun:"pro_number"`
	BOL        string   `bun:"bol"`
}

type candidateOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func TestSyncRecordRepository_ListCandidatesFindsPostedDocumentsWithoutARecord(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	_, err := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org candidateOrg
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))
	var userID pulid.ID
	require.NoError(t, db.NewSelect().
		Table("users").
		Column("id").
		Where("current_organization_id = ?", org.ID).
		Limit(1).
		Scan(ctx, &userID))
	shipments := make([]candidateShipment, 0, 3)
	require.NoError(t, db.NewSelect().
		Table("shipments").
		Column("id", "customer_id", "pro_number", "bol").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("id").
		Limit(3).
		Scan(ctx, &shipments))
	require.Len(t, shipments, 3)

	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoices := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	records := NewSyncRecordRepository(SyncRecordParams{DB: conn, Logger: logger})
	connection, err := NewConnectionRepository(ConnectionParams{DB: conn, Logger: logger}).
		Create(ctx, newConnection(tenant, userID, realm, 1_000))
	require.NoError(t, err)

	const (
		start   = int64(1_780_000_000)
		enabled = int64(1_790_000_000)
	)

	createInvoice := func(idx int, number string, status invoice.Status, dated int64, posted *int64) *invoice.Invoice {
		t.Helper()
		shp := shipments[idx]
		queue := &billingqueue.BillingQueueItem{
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			ShipmentID:       shp.ID,
			BillToCustomerID: shp.CustomerID,
			Number:           number,
			Status:           billingqueue.StatusPosted,
			BillType:         billingqueue.BillTypeInvoice,
		}
		_, insertErr := db.NewInsert().Model(queue).Exec(ctx)
		require.NoError(t, insertErr)
		created, createErr := invoices.Create(ctx, &invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             status,
			PostedAt:           posted,
			PaymentTerm:        invoice.PaymentTermNet30,
			CurrencyCode:       "USD",
			InvoiceDate:        dated,
			ShipmentProNumber:  shp.ProNumber,
			ShipmentBOL:        shp.BOL,
			BillToName:         "Test Customer",
			SubtotalAmount:     decimal.NewFromInt(100),
			OtherAmount:        decimal.Zero,
			TotalAmount:        decimal.NewFromInt(100),
			AppliedAmount:      decimal.Zero,
			SettlementStatus:   invoice.SettlementStatusUnpaid,
			DisputeStatus:      invoice.DisputeStatusNone,
		})
		require.NoError(t, createErr)
		return created
	}

	postedEarly := start + 100
	postedLate := enabled + 100
	backfillable := createInvoice(0, "INV-CAND-1", invoice.StatusPosted, start+50, &postedEarly)
	createInvoice(1, "INV-CAND-2", invoice.StatusPosted, start-86_400, &postedEarly)
	live := createInvoice(2, "INV-CAND-3", invoice.StatusPosted, enabled+10, &postedLate)

	list := func(req *repositories.ListAccountingSyncCandidatesRequest) []repositories.AccountingSyncCandidate {
		t.Helper()
		req.TenantInfo = tenant
		req.ConnectionID = connection.ID
		req.DatedFrom = start
		candidates, listErr := records.ListCandidates(ctx, req)
		require.NoError(t, listErr)
		return candidates
	}
	ids := func(candidates []repositories.AccountingSyncCandidate) []pulid.ID {
		out := make([]pulid.ID, 0, len(candidates))
		for _, candidate := range candidates {
			out = append(out, candidate.ObjectID)
		}
		return out
	}

	before := enabled
	backfill := list(&repositories.ListAccountingSyncCandidatesRequest{
		ObjectType:   accountingsync.SyncObjectInvoice,
		Operation:    accountingsync.SyncOperationCreate,
		PostedBefore: &before,
	})
	assert.Equal(t, []pulid.ID{backfillable.ID}, ids(backfill), "dated before the start date is never sent")
	assert.Equal(t, "INV-CAND-1", backfill[0].ObjectNumber)
	assert.Equal(t, start+50, backfill[0].DocumentDate)
	assert.Equal(t, postedEarly, backfill[0].PostedAt)

	from := enabled
	safetyNet := list(&repositories.ListAccountingSyncCandidatesRequest{
		ObjectType: accountingsync.SyncObjectInvoice,
		Operation:  accountingsync.SyncOperationCreate,
		PostedFrom: &from,
	})
	assert.Equal(t, []pulid.ID{live.ID}, ids(safetyNet))

	assert.Empty(t, list(&repositories.ListAccountingSyncCandidatesRequest{
		ObjectType: accountingsync.SyncObjectInvoice,
		Operation:  accountingsync.SyncOperationCreate,
		AfterAt:    postedLate,
		AfterID:    live.ID,
	}), "the cursor resumes after the last document it saw")

	_, err = records.Enqueue(ctx, []*accountingsync.AccountingSyncRecord{
		accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
			TenantInfo:   tenant,
			ConnectionID: connection.ID,
			Key: accountingsync.SyncRecordKey{
				ObjectType: accountingsync.SyncObjectInvoice,
				ObjectID:   live.ID,
				Operation:  accountingsync.SyncOperationCreate,
			},
			SourceEvent: accountingsync.SyncSourceInvoicePosted,
			At:          postedLate,
		}),
	})
	require.NoError(t, err)
	assert.Empty(t, list(&repositories.ListAccountingSyncCandidatesRequest{
		ObjectType: accountingsync.SyncObjectInvoice,
		Operation:  accountingsync.SyncOperationCreate,
		PostedFrom: &from,
	}), "a document with a record is not found again")

	reversedAt := enabled - 10
	undone := &customerpayment.Payment{
		ID:                 pulid.MustNew("cpay_"),
		OrganizationID:     org.ID,
		BusinessUnitID:     org.BusinessUnitID,
		CustomerID:         shipments[0].CustomerID,
		PaymentDate:        start + 10,
		AccountingDate:     start + 10,
		AmountMinor:        5_000,
		AppliedAmountMinor: 0,
		Status:             customerpayment.StatusReversed,
		PaymentMethod:      customerpayment.MethodACH,
		ReferenceNumber:    "ACH-UNDONE",
		CurrencyCode:       "USD",
		ReversedAt:         &reversedAt,
		ReversedByID:       userID,
		CreatedByID:        userID,
		CreatedAt:          start + 20,
		UpdatedAt:          start + 20,
	}
	kept := &customerpayment.Payment{
		ID:                   pulid.MustNew("cpay_"),
		OrganizationID:       org.ID,
		BusinessUnitID:       org.BusinessUnitID,
		CustomerID:           shipments[0].CustomerID,
		PaymentDate:          start + 30,
		AccountingDate:       start + 30,
		AmountMinor:          7_500,
		UnappliedAmountMinor: 7_500,
		Status:               customerpayment.StatusPosted,
		PaymentMethod:        customerpayment.MethodCheck,
		CurrencyCode:         "USD",
		CreatedByID:          userID,
		CreatedAt:            start + 40,
		UpdatedAt:            start + 40,
	}
	_, err = db.NewInsert().Model(&[]*customerpayment.Payment{undone, kept}).Exec(ctx)
	require.NoError(t, err)
	paymentCols := buncolgen.PaymentColumns
	for _, payment := range []*customerpayment.Payment{undone, kept} {
		_, err = db.NewUpdate().
			Model((*customerpayment.Payment)(nil)).
			Set(paymentCols.CreatedAt.Set(), payment.PaymentDate+10).
			Where(paymentCols.ID.Eq(), payment.ID).
			Exec(ctx)
		require.NoError(t, err)
	}

	payments := list(&repositories.ListAccountingSyncCandidatesRequest{
		ObjectType:   accountingsync.SyncObjectCustomerPayment,
		Operation:    accountingsync.SyncOperationCreate,
		PostedBefore: &before,
		UndoneBefore: &before,
	})
	assert.Equal(t, []pulid.ID{kept.ID}, ids(payments), "a payment reversed before sync began nets to nothing")
	assert.Empty(t, payments[0].ObjectNumber)

	voids := list(&repositories.ListAccountingSyncCandidatesRequest{
		ObjectType: accountingsync.SyncObjectCustomerPayment,
		Operation:  accountingsync.SyncOperationVoid,
	})
	assert.Equal(t, []pulid.ID{undone.ID}, ids(voids))
	assert.Equal(t, "ACH-UNDONE", voids[0].ObjectNumber)
	assert.Equal(t, reversedAt, voids[0].PostedAt)

	_, err = records.ListCandidates(ctx, &repositories.ListAccountingSyncCandidatesRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		ObjectType:   accountingsync.SyncObjectCustomer,
		Operation:    accountingsync.SyncOperationCreate,
	})
	assert.Error(t, err, "customers are dependencies, never swept")

	stranger := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: org.BusinessUnitID}
	none, err := records.ListCandidates(ctx, &repositories.ListAccountingSyncCandidatesRequest{
		TenantInfo:   stranger,
		ConnectionID: connection.ID,
		ObjectType:   accountingsync.SyncObjectInvoice,
		Operation:    accountingsync.SyncOperationCreate,
	})
	require.NoError(t, err)
	assert.Empty(t, none)
}
