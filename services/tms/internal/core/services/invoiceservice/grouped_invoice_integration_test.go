//go:build integration

package invoiceservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/billingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/billingqueuerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/orderrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
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
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

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
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	customerRepo := mocks.NewMockCustomerRepository(t)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	var org seededInvoiceOrg
	require.NoError(t, db.NewSelect().
		Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededInvoiceUser
	require.NoError(t, db.NewSelect().
		Table("users").Column("id").
		Where("current_organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Limit(1).Scan(ctx, &user))

	// Two seeded shipments become the legs of a new order.
	var legRows []seededInvoiceShipment
	require.NoError(t, db.NewSelect().
		Table("shipments").Column("id", "customer_id", "pro_number", "bol").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Limit(2).Scan(ctx, &legRows))
	require.Len(t, legRows, 2, "need two seeded shipments")
	customerID := legRows[0].CustomerID

	// Accounting scaffolding so Post can create a journal posting.
	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.ReconciliationMode = tenant.ReconciliationModeDisabled
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AutoPostSourceEvents = []tenant.JournalSourceEventType{
		tenant.JournalSourceEventInvoicePosted,
	}
	control.DefaultARAccountID = lookupInvoiceGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.DefaultRevenueAccountID = lookupInvoiceGLAccount(
		t, ctx, db, org.ID, org.BusinessUnitID, "4000",
	)
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{
		OrganizationID:        org.ID,
		BusinessUnitID:        org.BusinessUnitID,
		Status:                fiscalyear.StatusOpen,
		Year:                  now.Year(),
		Name:                  fmt.Sprintf("FY %d", now.Year()),
		StartDate:             now.Add(-24 * time.Hour).Unix(),
		EndDate:               now.Add(24 * time.Hour).Unix(),
		IsCurrent:             true,
		AllowAdjustingEntries: true,
	})
	require.NoError(t, err)
	_, err = fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
		OrganizationID:        org.ID,
		BusinessUnitID:        org.BusinessUnitID,
		FiscalYearID:          fy.ID,
		PeriodNumber:          1,
		PeriodType:            fiscalperiod.PeriodTypeMonth,
		Status:                fiscalperiod.StatusOpen,
		Name:                  now.Format("January 2006"),
		StartDate:             now.Add(-24 * time.Hour).Unix(),
		EndDate:               now.Add(24 * time.Hour).Unix(),
		AllowAdjustingEntries: true,
	})
	require.NoError(t, err)

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
	for _, leg := range legRows {
		_, err = db.NewUpdate().
			Table("shipments").
			Set("order_id = ?", ord.ID).
			Set("customer_id = ?", customerID).
			Set("status = ?", shipment.StatusReadyToInvoice).
			Set("freight_charge_amount = ?", freights[leg.ID]).
			Where("id = ?", leg.ID).
			Exec(ctx)
		require.NoError(t, err)
	}

	// The service loads each leg's full detail through the shipment repository; return a
	// shipment carrying the freight amount used to build the invoice line.
	for _, leg := range legRows {
		legID := leg.ID
		proNumber := leg.ProNumber
		shipmentRepo.EXPECT().
			GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
				return req != nil && req.ID == legID
			})).
			Return(&shipment.Shipment{
				ID:                  legID,
				OrganizationID:      org.ID,
				BusinessUnitID:      org.BusinessUnitID,
				CustomerID:          customerID,
				Status:              shipment.StatusReadyToInvoice,
				ProNumber:           proNumber,
				FreightChargeAmount: decimal.NewNullDecimal(freights[legID]),
				TotalChargeAmount:   decimal.NewNullDecimal(freights[legID]),
			}, nil)
	}

	// Post marks every billed leg Invoiced.
	invoicedLegs := make(map[pulid.ID]bool)
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
			return entity != nil && entity.Status == shipment.StatusInvoiced && entity.BilledAt != nil
		})).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			invoicedLegs[entity.ID] = true
			return entity, nil
		})

	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetCustomerByIDRequest) bool {
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
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  org.ID,
		BuID:   org.BusinessUnitID,
		UserID: user.ID,
	}
	actor := testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID)

	// --- Create the grouped invoice from the order ---
	inv, err := svc.CreateFromOrder(ctx, &serviceports.CreateInvoiceFromOrderRequest{
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
	var queueItems []struct {
		ShipmentID pulid.ID `bun:"shipment_id"`
		OrderID    pulid.ID `bun:"order_id"`
		Status     string   `bun:"status"`
	}
	require.NoError(t, db.NewSelect().
		Table("billing_queue_items").
		Column("shipment_id", "order_id", "status").
		Where("order_id = ?", ord.ID).
		Scan(ctx, &queueItems))
	require.Len(t, queueItems, 2, "one billing queue item per leg")
	for _, qi := range queueItems {
		assert.Equal(t, ord.ID, qi.OrderID)
		assert.Equal(t, "Approved", qi.Status)
	}

	// --- Post the grouped invoice ---
	posted, err := svc.Post(ctx, &serviceports.PostInvoiceRequest{
		InvoiceID:  inv.ID,
		TenantInfo: tenantInfo,
	}, actor)
	require.NoError(t, err)
	require.Equal(t, "Posted", string(posted.Status))

	// Every leg's billing-queue item settled, not just the anchor.
	var postedCount int
	require.NoError(t, db.NewSelect().
		Table("billing_queue_items").
		ColumnExpr("count(*)").
		Where("order_id = ?", ord.ID).
		Where("status = ?", "Posted").
		Scan(ctx, &postedCount))
	assert.Equal(t, 2, postedCount, "both leg billing-queue items should be Posted")

	// Every leg was marked Invoiced.
	assert.True(t, invoicedLegs[legRows[0].ID], "leg 0 should be invoiced")
	assert.True(t, invoicedLegs[legRows[1].ID], "leg 1 should be invoiced")
}
