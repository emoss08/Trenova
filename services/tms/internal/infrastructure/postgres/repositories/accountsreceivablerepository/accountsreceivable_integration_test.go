//go:build integration

package accountsreceivablerepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/customerpaymentservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerledgerrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerpaymentrepository"
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
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type seededAROrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}
type seededARUser struct {
	ID pulid.ID `bun:"id"`
}
type seededARShipment struct {
	ID         pulid.ID `bun:"id"`
	CustomerID pulid.ID `bun:"customer_id"`
	ProNumber  string   `bun:"pro_number"`
	BOL        string   `bun:"bol"`
}

func TestAccountsReceivableRepositoryReturnsLedgerAndAging(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	repo := New(Params{DB: conn, Logger: logger})
	paymentRepo := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	ledgerProjectionRepo := customerledgerrepository.New(customerledgerrepository.Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededAROrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededARUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededARShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupARGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1010")
	control.DefaultUnappliedCashAccountID = lookupARGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "2200")
	control.DefaultARAccountID = lookupARGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: now.Year(), Name: "FY", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-AR-1", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64PtrAR(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(100), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(100), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)}}})
	require.NoError(t, err)
	require.NoError(t, ledgerProjectionRepo.AppendEntries(ctx, []*customerledger.Entry{{ID: pulid.MustNew("cledg_"), OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, CustomerID: shp.CustomerID, SourceObjectType: "Invoice", SourceObjectID: inv.ID.String(), SourceEventType: tenant.JournalSourceEventInvoicePosted.String(), RelatedInvoiceID: inv.ID, DocumentNumber: inv.Number, TransactionDate: postedAt, LineNumber: 1, AmountMinor: inv.TotalAmountMinor, CreatedByID: user.ID}}))

	paymentSvc := customerpaymentservice.New(customerpaymentservice.Params{Logger: logger, DB: conn, Repo: paymentRepo, InvoiceRepo: invoiceRepo, CustomerLedgerRepo: ledgerProjectionRepo, AccountingRepo: accountingRepo, JournalRepo: journalRepo, Generator: generator, Validator: customerpaymentservice.NewValidator(customerpaymentservice.ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	_, err = paymentSvc.PostAndApply(ctx, &serviceports.PostCustomerPaymentRequest{CustomerID: shp.CustomerID, PaymentDate: now.Unix(), AccountingDate: now.Unix(), AmountMinor: 15000, PaymentMethod: customerpayment.MethodACH, ReferenceNumber: "PAY-AR", CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv.ID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)

	ledger, err := repo.ListCustomerLedger(ctx, repositories.ListCustomerLedgerRequest{TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}, CustomerID: shp.CustomerID})
	require.NoError(t, err)
	require.Len(t, ledger, 2)
	amounts := []int64{ledger[0].AmountMinor, ledger[1].AmountMinor}
	assert.Contains(t, amounts, int64(10000))
	assert.Contains(t, amounts, int64(-10000))

	aging, err := repo.ListARAging(ctx, repositories.ListARAgingRequest{TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}, AsOfDate: now.Unix()})
	require.NoError(t, err)
	require.Len(t, aging, 0)
}

func lookupARGLAccount(t *testing.T, ctx context.Context, db *bun.DB, orgID, buID pulid.ID, accountCode string) pulid.ID {
	t.Helper()
	var row struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(t, db.NewSelect().Table("gl_accounts").Column("id").Where("organization_id = ?", orgID).Where("business_unit_id = ?", buID).Where("account_code = ?", accountCode).Limit(1).Scan(ctx, &row))
	return row.ID
}

func int64PtrAR(v int64) *int64 { return &v }
