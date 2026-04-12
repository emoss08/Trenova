//go:build integration

package customerpaymentservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
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

type seededPaymentOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}
type seededPaymentUser struct {
	ID pulid.ID `bun:"id"`
}
type seededPaymentShipment struct {
	ID         pulid.ID `bun:"id"`
	CustomerID pulid.ID `bun:"customer_id"`
	ProNumber  string   `bun:"pro_number"`
	BOL        string   `bun:"bol"`
}

func TestPostAndApplyCreatesPaymentApplicationJournalAndSettlementUpdate(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededPaymentOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededPaymentUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededPaymentShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1010")
	control.DefaultUnappliedCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "2200")
	control.DefaultARAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.DefaultRevenueAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "4000")
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AccountingBasis = tenant.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: now.Year(), Name: "FY", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-PAY-1", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64Ptr(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(100), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(100), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)}}})
	require.NoError(t, err)

	svc := New(Params{Logger: logger, DB: conn, Repo: paymentRepo, InvoiceRepo: invoiceRepo, AccountingRepo: accountingRepo, JournalRepo: journalRepo, Generator: generator, Validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	payment, err := svc.PostAndApply(ctx, &serviceports.PostCustomerPaymentRequest{CustomerID: shp.CustomerID, PaymentDate: now.Unix(), AccountingDate: now.Unix(), AmountMinor: 10000, PaymentMethod: customerpayment.MethodACH, ReferenceNumber: "PAY-1", CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv.ID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(10000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(0), payment.UnappliedAmountMinor)

	updatedInvoice, err := invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}})
	require.NoError(t, err)
	assert.Equal(t, int64(10000), updatedInvoice.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice.SettlementStatus)

	var applicationCount int
	applicationCount, err = db.NewSelect().Table("customer_payment_applications").Where("customer_payment_id = ?", payment.ID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, applicationCount)

	var entry struct {
		ReferenceType string `bun:"reference_type"`
		TotalDebit    int64  `bun:"total_debit"`
		TotalCredit   int64  `bun:"total_credit"`
	}
	require.NoError(t, db.NewSelect().Table("journal_entries").Column("reference_type", "total_debit", "total_credit").Where("reference_id = ?", payment.ID.String()).Limit(1).Scan(ctx, &entry))
	assert.Equal(t, tenant.JournalSourceEventCustomerPaymentPosted.String(), entry.ReferenceType)
	assert.Equal(t, int64(10000), entry.TotalDebit)
	assert.Equal(t, int64(10000), entry.TotalCredit)

	var sourceCount int
	sourceCount, err = db.NewSelect().Table("journal_sources").Where("source_object_id = ?", payment.ID.String()).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, sourceCount)
}

func TestPostAndApplySupportsUnappliedCash(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededPaymentOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededPaymentUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededPaymentShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1010")
	control.DefaultUnappliedCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "2200")
	control.DefaultARAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AccountingBasis = tenant.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: now.Year(), Name: "FY", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-PAY-2", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64Ptr(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(100), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(100), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)}}})
	require.NoError(t, err)

	svc := New(Params{Logger: logger, DB: conn, Repo: paymentRepo, InvoiceRepo: invoiceRepo, AccountingRepo: accountingRepo, JournalRepo: journalRepo, Generator: generator, Validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	payment, err := svc.PostAndApply(ctx, &serviceports.PostCustomerPaymentRequest{CustomerID: shp.CustomerID, PaymentDate: now.Unix(), AccountingDate: now.Unix(), AmountMinor: 15000, PaymentMethod: customerpayment.MethodACH, ReferenceNumber: "PAY-2", CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv.ID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(10000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(5000), payment.UnappliedAmountMinor)

	updatedInvoice, err := invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}})
	require.NoError(t, err)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice.SettlementStatus)

	var lineCount int
	lineCount, err = db.NewSelect().Table("journal_entry_lines").Where("journal_entry_id = (SELECT id FROM journal_entries WHERE reference_id = ? LIMIT 1)", payment.ID.String()).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, lineCount)

	var unappliedBalance struct {
		PeriodCreditMinor int64 `bun:"period_credit_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("period_credit_minor").Where("gl_account_id = ?", control.DefaultUnappliedCashAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &unappliedBalance))
	assert.Equal(t, int64(5000), unappliedBalance.PeriodCreditMinor)
}

func TestApplyUnappliedLaterCreatesReclassificationEntry(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededPaymentOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededPaymentUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededPaymentShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1010")
	control.DefaultUnappliedCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "2200")
	control.DefaultARAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AccountingBasis = tenant.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: now.Year(), Name: "FY", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)

	queue1 := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-PAY-3", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	queue2 := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-PAY-4", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	_, err = db.NewInsert().Model(queue1).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(queue2).Exec(ctx)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv1, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue1.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue1.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64Ptr(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(100), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(100), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)}}})
	require.NoError(t, err)
	inv2, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue2.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue2.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64Ptr(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(50), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(50), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50), Amount: decimal.NewFromInt(50)}}})
	require.NoError(t, err)

	svc := New(Params{Logger: logger, DB: conn, Repo: paymentRepo, InvoiceRepo: invoiceRepo, AccountingRepo: accountingRepo, JournalRepo: journalRepo, Generator: generator, Validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	payment, err := svc.PostAndApply(ctx, &serviceports.PostCustomerPaymentRequest{CustomerID: shp.CustomerID, PaymentDate: now.Unix(), AccountingDate: now.Unix(), AmountMinor: 15000, PaymentMethod: customerpayment.MethodACH, ReferenceNumber: "PAY-3", CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv1.ID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	assert.Equal(t, int64(5000), payment.UnappliedAmountMinor)

	payment, err = svc.ApplyUnapplied(ctx, &serviceports.ApplyCustomerPaymentRequest{PaymentID: payment.ID, AccountingDate: now.Unix(), Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv2.ID, AppliedAmountMinor: 5000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	assert.Equal(t, int64(15000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(0), payment.UnappliedAmountMinor)

	updatedInvoice2, err := invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: inv2.ID, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}})
	require.NoError(t, err)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice2.SettlementStatus)

	var appliedEntryCount int
	appliedEntryCount, err = db.NewSelect().Table("journal_entries").Where("reference_type = ?", "CustomerPaymentApplied").Where("reference_id = ?", payment.ID.String()).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, appliedEntryCount)

	var unappliedBalance struct {
		NetChangeMinor int64 `bun:"net_change_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("net_change_minor").Where("gl_account_id = ?", control.DefaultUnappliedCashAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &unappliedBalance))
	assert.Equal(t, int64(0), unappliedBalance.NetChangeMinor)
}

func TestReversePaymentRestoresInvoiceAndBalances(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededPaymentOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededPaymentUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededPaymentShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1010")
	control.DefaultUnappliedCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "2200")
	control.DefaultARAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AccountingBasis = tenant.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: now.Year(), Name: "FY", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-PAY-5", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64Ptr(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(100), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(100), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)}}})
	require.NoError(t, err)

	svc := New(Params{Logger: logger, DB: conn, Repo: paymentRepo, InvoiceRepo: invoiceRepo, AccountingRepo: accountingRepo, JournalRepo: journalRepo, Generator: generator, Validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	payment, err := svc.PostAndApply(ctx, &serviceports.PostCustomerPaymentRequest{CustomerID: shp.CustomerID, PaymentDate: now.Unix(), AccountingDate: now.Unix(), AmountMinor: 15000, PaymentMethod: customerpayment.MethodACH, ReferenceNumber: "PAY-REV", CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv.ID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)

	reversed, err := svc.Reverse(ctx, &serviceports.ReverseCustomerPaymentRequest{PaymentID: payment.ID, AccountingDate: now.Unix(), Reason: "chargeback", TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	assert.Equal(t, customerpayment.StatusReversed, reversed.Status)

	updatedInvoice, err := invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}})
	require.NoError(t, err)
	assert.Equal(t, int64(0), updatedInvoice.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusUnpaid, updatedInvoice.SettlementStatus)

	var reversalEntryCount int
	reversalEntryCount, err = db.NewSelect().Table("journal_entries").Where("reference_type = ?", tenant.JournalSourceEventCustomerPaymentReversed.String()).Where("reference_id = ?", payment.ID.String()).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, reversalEntryCount)

	var arBalance struct {
		NetChangeMinor int64 `bun:"net_change_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("net_change_minor").Where("gl_account_id = ?", control.DefaultARAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &arBalance))
	assert.Equal(t, int64(0), arBalance.NetChangeMinor)

	var cashBalance struct {
		NetChangeMinor int64 `bun:"net_change_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("net_change_minor").Where("gl_account_id = ?", control.DefaultCashAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &cashBalance))
	assert.Equal(t, int64(0), cashBalance.NetChangeMinor)
}

func TestPostAndApplyRecognizesShortPayAndSettlesInvoice(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(customerpaymentrepository.Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededPaymentOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededPaymentUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var shp seededPaymentShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "customer_id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1010")
	control.DefaultUnappliedCashAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "2200")
	control.DefaultARAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "1110")
	control.DefaultWriteOffAccountID = lookupPaymentGLAccount(t, ctx, db, org.ID, org.BusinessUnitID, "6940")
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.AccountingBasis = tenant.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: now.Year(), Name: "FY", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period", StartDate: now.Add(-24 * time.Hour).Unix(), EndDate: now.Add(24 * time.Hour).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, ShipmentID: shp.ID, Number: "INV-SP-1", Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice}
	_, err = db.NewInsert().Model(queue).Exec(ctx)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(ctx, &invoice.Invoice{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BillingQueueItemID: queue.ID, ShipmentID: shp.ID, CustomerID: shp.CustomerID, Number: queue.Number, BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &postedAt, PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: period.StartDate, DueDate: int64Ptr(period.EndDate), ShipmentProNumber: shp.ProNumber, ShipmentBOL: shp.BOL, BillToName: "Test Customer", SubtotalAmount: decimal.NewFromInt(100), OtherAmount: decimal.Zero, TotalAmount: decimal.NewFromInt(100), AppliedAmount: decimal.Zero, SettlementStatus: invoice.SettlementStatusUnpaid, DisputeStatus: invoice.DisputeStatusNone, Lines: []*invoice.Line{{LineNumber: 1, Type: invoice.LineTypeFreight, Description: "Freight", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)}}})
	require.NoError(t, err)

	svc := New(Params{Logger: logger, DB: conn, Repo: paymentRepo, InvoiceRepo: invoiceRepo, AccountingRepo: accountingRepo, JournalRepo: journalRepo, Generator: generator, Validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	payment, err := svc.PostAndApply(ctx, &serviceports.PostCustomerPaymentRequest{CustomerID: shp.CustomerID, PaymentDate: now.Unix(), AccountingDate: now.Unix(), AmountMinor: 9000, PaymentMethod: customerpayment.MethodACH, ReferenceNumber: "PAY-SP", CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: inv.ID, AppliedAmountMinor: 9000, ShortPayAmountMinor: 1000}}, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}}, testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID))
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(9000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(0), payment.UnappliedAmountMinor)

	updatedInvoice, err := invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}})
	require.NoError(t, err)
	assert.Equal(t, int64(10000), updatedInvoice.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice.SettlementStatus)

	var writeOffBalance struct {
		PeriodDebitMinor int64 `bun:"period_debit_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("period_debit_minor").Where("gl_account_id = ?", control.DefaultWriteOffAccountID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &writeOffBalance))
	assert.Equal(t, int64(1000), writeOffBalance.PeriodDebitMinor)
}

func lookupPaymentGLAccount(t *testing.T, ctx context.Context, db *bun.DB, orgID, buID pulid.ID, accountCode string) pulid.ID {
	t.Helper()

	var row struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(t, db.NewSelect().Table("gl_accounts").Column("id").Where("organization_id = ?", orgID).Where("business_unit_id = ?", buID).Where("account_code = ?", accountCode).Limit(1).Scan(ctx, &row))
	return row.ID
}

func int64Ptr(v int64) *int64 { return &v }
