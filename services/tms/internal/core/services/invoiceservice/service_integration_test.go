//go:build integration

package invoiceservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
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
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/billingqueuerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
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
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type seededInvoiceOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type seededInvoiceUser struct {
	ID pulid.ID `bun:"id"`
}

type seededInvoiceShipment struct {
	ID         pulid.ID `bun:"id"`
	CustomerID pulid.ID `bun:"customer_id"`
	ProNumber  string   `bun:"pro_number"`
	BOL        string   `bun:"bol"`
}

func TestPostCreatesInvoiceJournalSourceAndBalances(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	billingQueueRepo := billingqueuerepository.New(billingqueuerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededInvoiceOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededInvoiceUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededInvoiceShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.ReconciliationMode = tenant.ReconciliationModeDisabled
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AutoPostSourceEvents = []tenant.JournalSourceEventType{tenant.JournalSourceEventInvoicePosted}
	control.DefaultARAccountID = lookupInvoiceGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.DefaultRevenueAccountID = lookupInvoiceGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "4000")
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
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
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

	queue := &billingqueue.BillingQueueItem{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		ShipmentID:     shp.ID,
		Number:         "INV-1001",
		Status:         billingqueue.StatusApproved,
		BillType:       billingqueue.BillTypeInvoice,
	}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)

	entity, err := invoiceRepo.Create(ctx, &invoice.Invoice{
		OrganizationID:     org.ID,
		BusinessUnitID:     org.BusinessUnitID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shp.ID,
		CustomerID:         shp.CustomerID,
		Number:             queue.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        period.StartDate,
		DueDate:            int64Ptr(period.EndDate),
		ShipmentProNumber:  shp.ProNumber,
		ShipmentBOL:        shp.BOL,
		BillToName:         "Test Customer",
		SubtotalAmount:     decimal.NewFromInt(100),
		OtherAmount:        decimal.Zero,
		TotalAmount:        decimal.NewFromInt(100),
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines: []*invoice.Line{{
			LineNumber:  1,
			Type:        invoice.LineTypeFreight,
			Description: "Freight",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(100),
			Amount:      decimal.NewFromInt(100),
		}},
	})
	require.NoError(t, err)

	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
		return req != nil && req.ID == shp.ID && req.TenantInfo.OrgID == org.ID && req.TenantInfo.BuID == org.BusinessUnitID
	})).Return(&shipment.Shipment{
		ID:             shp.ID,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Status:         shipment.StatusReadyToInvoice,
	}, nil)
	shipmentRepo.EXPECT().UpdateDerivedState(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
		return entity != nil && entity.ID == shp.ID && entity.Status == shipment.StatusInvoiced && entity.BilledAt != nil
	})).Return(&shipment.Shipment{ID: shp.ID}, nil)

	svc := &Service{
		l:                 logger,
		db:                conn,
		repo:              invoiceRepo,
		billingQueueRepo:  billingQueueRepo,
		shipmentRepo:      shipmentRepo,
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		validator:         NewValidator(ValidatorParams{DB: conn, Logger: logger, AccountingRepo: accountingRepo, FiscalPeriodRepo: fiscalPeriodRepo, ShipmentRepo: shipmentRepo}),
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		sequenceGenerator: generator,
	}

	posted, err := svc.Post(ctx, &serviceports.PostInvoiceRequest{InvoiceID: entity.ID, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	require.Equal(t, invoice.StatusPosted, posted.Status)
	require.NotNil(t, posted.PostedAt)

	var entry struct {
		ReferenceType string `bun:"reference_type"`
		ReferenceID   string `bun:"reference_id"`
		TotalDebit    int64  `bun:"total_debit"`
		TotalCredit   int64  `bun:"total_credit"`
	}
	require.NoError(t, db.NewSelect().Table("journal_entries").Column("reference_type", "reference_id", "total_debit", "total_credit").Where("reference_id = ?", entity.ID.String()).Limit(1).Scan(ctx, &entry))
	assert.Equal(t, tenant.JournalSourceEventInvoicePosted.String(), entry.ReferenceType)
	assert.Equal(t, entity.ID.String(), entry.ReferenceID)
	assert.Equal(t, int64(10000), entry.TotalDebit)
	assert.Equal(t, int64(10000), entry.TotalCredit)

	var source struct {
		SourceEventType string `bun:"source_event_type"`
		Status          string `bun:"status"`
	}
	require.NoError(t, db.NewSelect().Table("journal_sources").Column("source_event_type", "status").Where("source_object_id = ?", entity.ID.String()).Limit(1).Scan(ctx, &source))
	assert.Equal(t, tenant.JournalSourceEventInvoicePosted.String(), source.SourceEventType)
	assert.Equal(t, "Posted", source.Status)

	var arBalance struct {
		PeriodDebitMinor int64 `bun:"period_debit_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("period_debit_minor").Where("gl_account_id = ?", control.DefaultARAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &arBalance))
	assert.Equal(t, int64(10000), arBalance.PeriodDebitMinor)

	var revenueBalance struct {
		PeriodCreditMinor int64 `bun:"period_credit_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("period_credit_minor").Where("gl_account_id = ?", control.DefaultRevenueAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &revenueBalance))
	assert.Equal(t, int64(10000), revenueBalance.PeriodCreditMinor)
}

func lookupInvoiceGLAccount(t *testing.T, ctx context.Context, db *bun.DB, orgID, buID pulid.ID, accountCode string) pulid.ID {
	t.Helper()

	var row struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(t, db.NewSelect().Table("gl_accounts").Column("id").Where("organization_id = ?", orgID).Where("business_unit_id = ?", buID).Where("account_code = ?", accountCode).Limit(1).Scan(ctx, &row))
	return row.ID
}
