//go:build integration

package customerpaymentservice

import (
	"slices"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededPaymentOrg = billingtest.Org
type seededPaymentUser = billingtest.User
type seededPaymentShipment = billingtest.Shipment

func TestPostAndApplyCreatesPaymentApplicationJournalAndSettlementUpdate(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shp := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.DefaultRevenueAccountID = billingtest.GLAccountID(t, ctx, db, "4000")
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, period := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-PAY-1",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
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
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
					Amount:      decimal.NewFromInt(100),
				},
			},
		},
	)
	require.NoError(t, err)

	svc := New(
		Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: NewValidator(
				ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	payment, err := svc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     10000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "PAY-1",
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv.ID, AppliedAmountMinor: 10000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(10000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(0), payment.UnappliedAmountMinor)

	updatedInvoice, err := invoiceRepo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{
			ID:         inv.ID,
			TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(10000), updatedInvoice.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice.SettlementStatus)

	apps := customerpaymentdb.Applications
	applicationCount, err := apps.Where(apps.CustomerPaymentID.Eq(payment.ID)).Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), applicationCount)

	je := journalentrydb.JournalEntries
	entry, err := je.Pick(je.ReferenceType, je.TotalDebit, je.TotalCredit).
		Where(je.ReferenceID.Eq(payment.ID.String())).
		Order(je.ID.Asc()).
		First(ctx, db)
	require.NoError(t, err)
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventCustomerPaymentPosted.String(),
		entry.ReferenceType,
	)
	assert.Equal(t, int64(10000), entry.TotalDebit)
	assert.Equal(t, int64(10000), entry.TotalCredit)

	js := journalsourcedb.Sources
	sourceCount, err := js.Where(js.SourceObjectID.Eq(payment.ID.String())).Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), sourceCount)
}

func TestPostAndApplySupportsUnappliedCash(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shp := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, period := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-PAY-2",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
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
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
					Amount:      decimal.NewFromInt(100),
				},
			},
		},
	)
	require.NoError(t, err)

	svc := New(
		Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: NewValidator(
				ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	payment, err := svc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     15000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "PAY-2",
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv.ID, AppliedAmountMinor: 10000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(10000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(5000), payment.UnappliedAmountMinor)

	updatedInvoice, err := invoiceRepo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{
			ID:         inv.ID,
			TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice.SettlementStatus)

	je := journalentrydb.JournalEntries
	jel := journalentrydb.JournalEntryLines
	lineCount, err := jel.Where(jel.JournalEntryID.InQuery(
		je.Pick(je.ID).Where(je.ReferenceID.Eq(payment.ID.String())).Limit(1),
	)).Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(3), lineCount)

	unappliedBalance := fiscaltest.PeriodBalance(t, ctx, fiscaltest.PeriodBalanceParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		GLAccountID:    control.DefaultUnappliedCashAccountID,
		FiscalPeriodID: period.ID,
	})
	assert.Equal(t, int64(5000), unappliedBalance.PeriodCreditMinor)
}

func TestApplyUnappliedLaterCreatesReclassificationEntry(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shp := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, period := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue1 := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-PAY-3",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	queue2 := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-PAY-4",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, queue1))
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, queue2))
	postedAt := now.Unix()
	inv1, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue1.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue1.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
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
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
					Amount:      decimal.NewFromInt(100),
				},
			},
		},
	)
	require.NoError(t, err)
	inv2, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue2.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue2.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
			PaymentTerm:        invoice.PaymentTermNet30,
			CurrencyCode:       "USD",
			InvoiceDate:        period.StartDate,
			DueDate:            int64Ptr(period.EndDate),
			ShipmentProNumber:  shp.ProNumber,
			ShipmentBOL:        shp.BOL,
			BillToName:         "Test Customer",
			SubtotalAmount:     decimal.NewFromInt(50),
			OtherAmount:        decimal.Zero,
			TotalAmount:        decimal.NewFromInt(50),
			AppliedAmount:      decimal.Zero,
			SettlementStatus:   invoice.SettlementStatusUnpaid,
			DisputeStatus:      invoice.DisputeStatusNone,
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(50),
					Amount:      decimal.NewFromInt(50),
				},
			},
		},
	)
	require.NoError(t, err)

	svc := New(
		Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: NewValidator(
				ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	payment, err := svc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     15000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "PAY-3",
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv1.ID, AppliedAmountMinor: 10000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), payment.UnappliedAmountMinor)

	payment, err = svc.ApplyUnapplied(
		ctx,
		&customerpaymentports.ApplyCustomerPaymentRequest{
			PaymentID:      payment.ID,
			AccountingDate: now.Unix(),
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv2.ID, AppliedAmountMinor: 5000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(15000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(0), payment.UnappliedAmountMinor)

	updatedInvoice2, err := invoiceRepo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{
			ID:         inv2.ID,
			TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice2.SettlementStatus)

	je := journalentrydb.JournalEntries
	appliedEntryCount, err := je.Where(
		je.ReferenceType.Eq("CustomerPaymentApplied"),
		je.ReferenceID.Eq(payment.ID.String()),
	).Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), appliedEntryCount)

	unappliedBalance := fiscaltest.PeriodBalance(t, ctx, fiscaltest.PeriodBalanceParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		GLAccountID:    control.DefaultUnappliedCashAccountID,
		FiscalPeriodID: period.ID,
	})
	assert.Equal(t, int64(0), unappliedBalance.NetChangeMinor)
}

func TestReversePaymentRestoresInvoiceAndBalances(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shp := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, period := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-PAY-5",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
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
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
					Amount:      decimal.NewFromInt(100),
				},
			},
		},
	)
	require.NoError(t, err)

	svc := New(
		Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: NewValidator(
				ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	balanceOf := func(accountID pulid.ID) glbalance.GLAccountPeriodBalance {
		t.Helper()
		return fiscaltest.PeriodBalance(t, ctx, fiscaltest.PeriodBalanceParams{
			DB:             db,
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			GLAccountID:    accountID,
			FiscalPeriodID: period.ID,
		})
	}
	arBefore := balanceOf(control.DefaultARAccountID)
	cashBefore := balanceOf(control.DefaultCashAccountID)

	payment, err := svc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     15000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "PAY-REV",
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv.ID, AppliedAmountMinor: 10000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)

	reversed, err := svc.Reverse(
		ctx,
		&customerpaymentports.ReverseCustomerPaymentRequest{
			PaymentID:      payment.ID,
			AccountingDate: now.Unix(),
			Reason:         "chargeback",
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	assert.Equal(t, customerpayment.StatusReversed, reversed.Status)

	updatedInvoice, err := invoiceRepo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{
			ID:         inv.ID,
			TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(0), updatedInvoice.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusUnpaid, updatedInvoice.SettlementStatus)

	je := journalentrydb.JournalEntries
	reversalEntryCount, err := je.Where(
		je.ReferenceType.Eq(accountingcontrol.JournalSourceEventCustomerPaymentReversed.String()),
		je.ReferenceID.Eq(payment.ID.String()),
	).Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), reversalEntryCount)

	assert.Equal(
		t,
		arBefore.NetChangeMinor,
		balanceOf(control.DefaultARAccountID).NetChangeMinor,
	)
	assert.Equal(
		t,
		cashBefore.NetChangeMinor,
		balanceOf(control.DefaultCashAccountID).NetChangeMinor,
	)
}

func TestPostAndApplyRecognizesShortPayAndSettlesInvoice(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shp := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.DefaultWriteOffAccountID = billingtest.GLAccountID(t, ctx, db, "6940")
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, period := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-SP-1",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
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
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
					Amount:      decimal.NewFromInt(100),
				},
			},
		},
	)
	require.NoError(t, err)

	svc := New(
		Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: NewValidator(
				ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	payment, err := svc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     9000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "PAY-SP",
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv.ID, AppliedAmountMinor: 9000, ShortPayAmountMinor: 1000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(9000), payment.AppliedAmountMinor)
	assert.Equal(t, int64(0), payment.UnappliedAmountMinor)

	updatedInvoice, err := invoiceRepo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{
			ID:         inv.ID,
			TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(10000), updatedInvoice.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPaid, updatedInvoice.SettlementStatus)

	writeOffBalance := fiscaltest.PeriodBalance(t, ctx, fiscaltest.PeriodBalanceParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		GLAccountID:    control.DefaultWriteOffAccountID,
		FiscalPeriodID: period.ID,
	})
	assert.Equal(t, int64(1000), writeOffBalance.PeriodDebitMinor)
}

func TestListAndGetCustomerPayments(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shp := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, period := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           "INV-PAY-LIST-1",
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             queue.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
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
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
					Amount:      decimal.NewFromInt(100),
				},
			},
		},
	)
	require.NoError(t, err)

	svc := New(
		Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: NewValidator(
				ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalPeriodRepo},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	payment, err := svc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     10000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "PAY-LIST-1",
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv.ID, AppliedAmountMinor: 10000},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)

	listResult, err := svc.List(
		ctx,
		&customerpaymentports.ListCustomerPaymentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID:  org.ID,
					BuID:   org.BusinessUnitID,
					UserID: user.ID,
				},
				Pagination: pagination.Info{Limit: 10, Offset: 0},
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, listResult)
	isPayment := func(item *customerpayment.Payment) bool { return item.ID == payment.ID }
	assert.True(t, slices.ContainsFunc(listResult.Items, isPayment))

	connResult, err := paymentRepo.ListConnection(
		ctx,
		&customerpaymentports.ListCustomerPaymentConnectionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID:  org.ID,
					BuID:   org.BusinessUnitID,
					UserID: user.ID,
				},
				Pagination: pagination.Info{Limit: 10, Offset: 0},
			},
			Cursor: pagination.CursorInfo{Limit: 10},
		},
	)
	require.NoError(t, err)
	connIdx := slices.IndexFunc(connResult.Items, isPayment)
	require.NotEqual(t, -1, connIdx)
	connPayment := connResult.Items[connIdx]
	assert.Equal(t, payment.ID, connPayment.ID)
	assert.Equal(t, shp.CustomerID, connPayment.CustomerID)
	assert.EqualValues(t, 10000, connPayment.AmountMinor)
	assert.Equal(t, customerpayment.StatusPosted, connPayment.Status)
	require.NotEmpty(t, connPayment.Applications)
	assert.Equal(t, inv.ID, connPayment.Applications[0].InvoiceID)

	got, err := svc.Get(
		ctx,
		&customerpaymentports.GetCustomerPaymentRequest{
			PaymentID: payment.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, payment.ID, got.ID)
}

func int64Ptr(v int64) *int64 { return &v }
