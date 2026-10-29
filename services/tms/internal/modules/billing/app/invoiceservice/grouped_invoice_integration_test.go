//go:build integration

package invoiceservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/billingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderderivation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestGroupedInvoiceFromOrderEndToEnd exercises the full grouped-invoicing path: an
// order with two billable legs produces one invoice (lines attributed per leg), one
// billing-queue item per leg, and posting the invoice settles every leg's queue item
// and marks every leg Invoiced — not just the anchor.
func TestGroupedInvoiceFromOrderEndToEnd(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	billingQueueRepo := billingqueuerepository.New(
		billingqueuerepository.Params{DB: conn, Logger: logger},
	)
	orderRepo := orderrepository.New(orderrepository.Params{DB: conn, Logger: logger})
	billingRepo := billingcontrolrepository.New(
		billingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	ctx, org, user, _ := seededInvoiceFixture(t, ctx, db)

	// Two seeded shipments become the legs of the grouped order; two more feed the
	// partial-billing scenario at the end.
	shipmentRows, err := db.Pool().Query(ctx,
		`SELECT id, customer_id, COALESCE(pro_number, ''), COALESCE(bol, '') FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY created_at ASC LIMIT 4`,
		org.ID, org.BusinessUnitID,
	)
	require.NoError(t, err)
	legRows, err := pgx.CollectRows(shipmentRows, pgx.RowToStructByPos[seededInvoiceShipment])
	require.NoError(t, err)
	require.Len(t, legRows, 4, "need four seeded shipments")
	customerID := legRows[0].CustomerID

	// Accounting scaffolding so Post can create a journal posting.
	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.ReconciliationMode = accountingcontrol.ReconciliationModeWarnOnly
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AutoPostSourceEvents = []accountingcontrol.JournalSourceEventType{
		accountingcontrol.JournalSourceEventInvoicePosted,
	}
	control.DefaultARAccountID = lookupInvoiceGLAccount(
		t,
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		"1110",
	)
	control.DefaultRevenueAccountID = lookupInvoiceGLAccount(
		t, ctx, db, org.ID, org.BusinessUnitID, "4000",
	)
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	// Create the order and attach the two legs to it.
	ord, err := orderRepo.Create(ctx, &order.Order{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		CustomerID:     customerID,
		Status:         order.StatusConfirmed,
		OrderNumber:    "O-TEST-0001",
		CurrencyCode:   "USD",
	})
	require.NoError(t, err)

	// An order-level charge (e.g. customs brokerage) must roll into the total and appear
	// as its own line on the grouped invoice.
	_, err = orderRepo.AddCharge(ctx, &order.OrderCharge{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		OrderID:        ord.ID,
		Description:    "Customs brokerage",
		Amount:         decimal.NewFromInt(75),
	})
	require.NoError(t, err)

	freights := map[pulid.ID]decimal.Decimal{
		legRows[0].ID: decimal.NewFromInt(100),
		legRows[1].ID: decimal.NewFromInt(250),
	}
	for _, leg := range legRows[:2] {
		setLeg(
			t,
			ctx,
			db,
			leg.ID,
			ord.ID,
			customerID,
			shipment.StatusReadyToInvoice,
			freights[leg.ID],
		)
	}

	// The service loads each leg's full detail through the shipment repository; return a
	// shipment carrying the freight amount used to build the invoice line.
	mockLeg := func(legID pulid.ID, proNumber string, orderID pulid.ID, freight decimal.Decimal) {
		shipmentRepo.EXPECT().
			GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
				return req != nil && req.ID == legID
			})).
			Return(&shipment.Shipment{
				ID:                  legID,
				OrganizationID:      org.ID,
				BusinessUnitID:      org.BusinessUnitID,
				CustomerID:          customerID,
				OrderID:             orderID,
				Status:              shipment.StatusReadyToInvoice,
				ProNumber:           proNumber,
				FreightChargeAmount: decimal.NewNullDecimal(freight),
				TotalChargeAmount:   decimal.NewNullDecimal(freight),
			}, nil)
	}
	for _, leg := range legRows[:2] {
		mockLeg(leg.ID, leg.ProNumber, ord.ID, freights[leg.ID])
	}

	// Post marks every billed leg Invoiced.
	invoicedLegs := make(map[pulid.ID]bool)
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
			return entity != nil && entity.Status == shipment.StatusInvoiced &&
				entity.BilledAt != nil
		})).
		RunAndReturn(func(runCtx context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			invoicedLegs[entity.ID] = true
			_, dbErr := keel.Exec(runCtx, db,
				"UPDATE shipments SET status = $1 WHERE id = $2",
				string(shipment.StatusInvoiced), entity.ID.String())
			return entity, dbErr
		})

	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
			return req.ID == customerID
		})).
		Return(&customer.Customer{
			ID:             customerID,
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Name:           "Grouped Customer",
			Code:           "GRP",
		}, nil)

	svc := &Service{
		l:                logger,
		db:               conn,
		repo:             invoiceRepo,
		billingQueueRepo: billingQueueRepo,
		shipmentRepo:     shipmentRepo,
		orderRepo:        orderRepo,
		customerRepo:     customerRepo,
		billingRepo:      billingRepo,
		accountingRepo:   accountingRepo,
		journalRepo:      journalRepo,
		validator: NewValidator(ValidatorParams{
			DB:               conn,
			Logger:           logger,
			AccountingRepo:   accountingRepo,
			FiscalPeriodRepo: fiscalPeriodRepo,
			ShipmentRepo:     shipmentRepo,
		}),
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		sequenceGenerator: generator,
		orderDerivation: orderderivation.New(orderderivation.Params{
			Logger:       logger,
			ShipmentRepo: shipmentRepo,
			OrderRepo:    orderRepo,
			Realtime:     &mocks.NoopRealtimeService{},
		}),
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  org.ID,
		BuID:   org.BusinessUnitID,
		UserID: user.ID,
	}
	actor := testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID)

	// --- Create the grouped invoice from the order ---
	inv, err := svc.CreateFromOrder(ctx, &invoiceports.CreateInvoiceFromOrderRequest{
		OrderID:    ord.ID,
		TenantInfo: tenantInfo,
	}, actor)
	require.NoError(t, err)
	require.NotNil(t, inv)

	assert.Equal(t, ord.ID, inv.OrderID, "invoice should belong to the order")
	assert.True(t, inv.ShipmentID.IsNil(), "grouped invoice header has no single shipment")
	assert.Equal(t, "O-TEST-0001", inv.OrderNumber)
	assert.True(t, decimal.NewFromInt(425).Equal(inv.TotalAmount),
		"total should sum both legs plus the order charge, got %s", inv.TotalAmount)

	// Each leg contributes a line attributed to it; the order charge is its own line
	// with no leg attribution.
	lineShipmentIDs := map[pulid.ID]bool{}
	foundCharge := false
	for _, line := range inv.Lines {
		if line.ShipmentID.IsNil() {
			if line.Description == "Customs brokerage" {
				foundCharge = true
			}
			continue
		}
		lineShipmentIDs[line.ShipmentID] = true
	}
	assert.Len(t, lineShipmentIDs, 2, "leg lines should cover both legs")
	assert.True(t, foundCharge, "grouped invoice should include the order-level charge line")

	// One approved billing-queue item per leg, all pointing at the order.
	type queueRow struct {
		ShipmentID pulid.ID
		OrderID    pulid.ID
		Status     string
	}
	queueRows, err := db.Pool().Query(ctx,
		"SELECT shipment_id, order_id, status::text FROM billing_queue_items WHERE order_id = $1",
		ord.ID,
	)
	require.NoError(t, err)
	queueItems, err := pgx.CollectRows(queueRows, pgx.RowToStructByPos[queueRow])
	require.NoError(t, err)
	require.Len(t, queueItems, 2, "one billing queue item per leg")
	for _, qi := range queueItems {
		assert.Equal(t, ord.ID, qi.OrderID)
		assert.Equal(t, "Approved", qi.Status)
	}

	// --- Post the grouped invoice ---
	posted, err := svc.Post(ctx, &invoiceports.PostInvoiceRequest{
		InvoiceID:  inv.ID,
		TenantInfo: tenantInfo,
	}, actor)
	require.NoError(t, err)
	require.Equal(t, "Posted", string(posted.Status))

	// Every leg's billing-queue item settled, not just the anchor.
	var postedCount int
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT count(*) FROM billing_queue_items WHERE order_id = $1 AND status = 'Posted'",
		ord.ID,
	).Scan(&postedCount))
	assert.Equal(t, 2, postedCount, "both leg billing-queue items should be Posted")

	// Each leg's shipment shows its billing-queue item as Posted.
	var postedLegCount int
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT count(*) FROM shipments WHERE id = ANY($1) AND billing_transfer_status = 'Posted'",
		[]string{legRows[0].ID.String(), legRows[1].ID.String()},
	).Scan(&postedLegCount))
	assert.Equal(t, 2, postedLegCount, "both leg shipments should show Posted")

	// Every leg was marked Invoiced.
	assert.True(t, invoicedLegs[legRows[0].ID], "leg 0 should be invoiced")
	assert.True(t, invoicedLegs[legRows[1].ID], "leg 1 should be invoiced")

	// Posting derived the order to Billed (all legs Invoiced) and the total held.
	billedOrder, err := orderRepo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID:         ord.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.Equal(t, order.StatusBilled, billedOrder.Status, "order should derive to Billed")
	assert.True(t, decimal.NewFromInt(425).Equal(billedOrder.TotalAmount.Decimal),
		"order total should match the invoice, got %s", billedOrder.TotalAmount.Decimal)

	// A second CreateFromOrder for the already-invoiced order must be rejected, not
	// silently duplicated.
	_, err = svc.CreateFromOrder(ctx, &invoiceports.CreateInvoiceFromOrderRequest{
		OrderID:    ord.ID,
		TenantInfo: tenantInfo,
	}, actor)
	require.Error(t, err, "double CreateFromOrder must conflict")

	runPartialOrderScenario(
		t, ctx, db, svc, orderRepo,
		partialScenarioParams{
			tenantInfo: tenantInfo,
			actor:      actor,
			org:        org,
			customerID: customerID,
			readyLeg:   legRows[2],
			lateLeg:    legRows[3],
			mockLeg:    mockLeg,
		},
	)
}

type partialScenarioParams struct {
	tenantInfo pagination.TenantInfo
	actor      *permissionports.RequestActor
	org        seededInvoiceOrg
	customerID pulid.ID
	readyLeg   seededInvoiceShipment
	lateLeg    seededInvoiceShipment
	mockLeg    func(pulid.ID, string, pulid.ID, decimal.Decimal)
}

// runPartialOrderScenario covers partial-order billing: only ready legs are invoiced,
// order-level charges are billed exactly once (first pass), and the follow-up
// single-leg invoice still carries the order correlation.
func runPartialOrderScenario(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	svc *Service,
	orderRepo orderports.OrderRepository,
	p partialScenarioParams,
) {
	t.Helper()

	ord, err := orderRepo.Create(ctx, &order.Order{
		OrganizationID: p.org.ID,
		BusinessUnitID: p.org.BusinessUnitID,
		CustomerID:     p.customerID,
		Status:         order.StatusInProgress,
		OrderNumber:    "O-TEST-0002",
		CurrencyCode:   "USD",
	})
	require.NoError(t, err)

	_, err = orderRepo.AddCharge(ctx, &order.OrderCharge{
		OrganizationID: p.org.ID,
		BusinessUnitID: p.org.BusinessUnitID,
		OrderID:        ord.ID,
		Description:    "Border handling",
		Amount:         decimal.NewFromInt(50),
	})
	require.NoError(t, err)

	legStates := map[pulid.ID]shipment.Status{
		p.readyLeg.ID: shipment.StatusReadyToInvoice,
		p.lateLeg.ID:  shipment.StatusInTransit,
	}
	legFreights := map[pulid.ID]decimal.Decimal{
		p.readyLeg.ID: decimal.NewFromInt(100),
		p.lateLeg.ID:  decimal.NewFromInt(200),
	}
	for legID, status := range legStates {
		setLeg(t, ctx, db, legID, ord.ID, p.customerID, status, legFreights[legID])
	}
	p.mockLeg(p.readyLeg.ID, p.readyLeg.ProNumber, ord.ID, legFreights[p.readyLeg.ID])
	p.mockLeg(p.lateLeg.ID, p.lateLeg.ProNumber, ord.ID, legFreights[p.lateLeg.ID])

	// First pass: only the ready leg is billable, and the order charge rides along.
	first, err := svc.CreateFromOrder(ctx, &invoiceports.CreateInvoiceFromOrderRequest{
		OrderID:    ord.ID,
		TenantInfo: p.tenantInfo,
	}, p.actor)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(150).Equal(first.TotalAmount),
		"first pass should bill the ready leg plus the order charge, got %s", first.TotalAmount)

	// The charge was stamped to the first invoice inside the same transaction.
	var stampedInvoiceID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT invoice_id FROM order_charges WHERE order_id = $1", ord.ID,
	).Scan(&stampedInvoiceID))
	assert.Equal(
		t,
		first.ID,
		stampedInvoiceID,
		"order charge should be stamped to the first invoice",
	)

	// The late leg delivers; the ready leg was settled by billing (simulated here since
	// the first invoice is left in Draft).
	_, err = db.Pool().Exec(ctx, "UPDATE shipments SET status = $1 WHERE id = $2",
		string(shipment.StatusInvoiced), p.readyLeg.ID)
	require.NoError(t, err)
	_, err = db.Pool().Exec(ctx, "UPDATE shipments SET status = $1 WHERE id = $2",
		string(shipment.StatusReadyToInvoice), p.lateLeg.ID)
	require.NoError(t, err)

	// Second pass through the single-shipment entry point: the invoice carries the
	// order correlation (G9) and must NOT re-bill the order charge.
	second, err := svc.CreateFromShipments(ctx, &invoiceports.CreateInvoiceFromShipmentsRequest{
		ShipmentIDs: []pulid.ID{p.lateLeg.ID},
		TenantInfo:  p.tenantInfo,
	}, p.actor)
	require.NoError(t, err)
	assert.Equal(t, ord.ID, second.OrderID, "single-leg invoice should keep the order correlation")
	assert.Equal(t, "O-TEST-0002", second.OrderNumber)
	assert.True(
		t,
		decimal.NewFromInt(200).Equal(second.TotalAmount),
		"second pass must bill only the leg — no duplicated order charge, got %s",
		second.TotalAmount,
	)
	for _, line := range second.Lines {
		assert.False(t, line.ShipmentID.IsNil(),
			"second invoice must not carry order-charge lines")
	}
}

func setLeg(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	legID, orderID, customerID pulid.ID,
	status shipment.Status,
	freight decimal.Decimal,
) {
	t.Helper()

	_, err := db.Pool().Exec(ctx,
		`UPDATE shipments SET order_id = $1, customer_id = $2, status = $3,
			freight_charge_amount = $4, total_charge_amount = $4, other_charge_amount = 0
		WHERE id = $5`,
		orderID, customerID, string(status), freight.String(), legID,
	)
	require.NoError(t, err)
}
