//go:build integration

package bankreceiptservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/customerpaymentservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededBROrg = billingtest.Org
type seededBRUser = billingtest.User
type seededBRShipment = billingtest.Shipment

func TestImportAndMatchBankReceiptToCustomerPayment(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	receiptRepo := bankreceiptrepository.New(bankreceiptrepository.Params{DB: conn, Logger: logger})
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
		Number:           "INV-BR-1",
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

	paymentSvc := customerpaymentservice.New(
		customerpaymentservice.Params{
			Logger:         logger,
			DB:             conn,
			Repo:           paymentRepo,
			InvoiceRepo:    invoiceRepo,
			AccountingRepo: accountingRepo,
			JournalRepo:    journalRepo,
			Generator:      generator,
			Validator: customerpaymentservice.NewValidator(
				customerpaymentservice.ValidatorParams{
					InvoiceRepo:      invoiceRepo,
					FiscalPeriodRepo: fiscalPeriodRepo,
				},
			),
			AuditService: &mocks.NoopAuditService{},
		},
	)
	payment, err := paymentSvc.PostAndApply(
		ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      shp.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     10000,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: "BANK-MATCH-1",
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

	svc := New(
		Params{
			Logger:       logger,
			DB:           conn,
			Repo:         receiptRepo,
			PaymentRepo:  paymentRepo,
			AuditService: &mocks.NoopAuditService{},
		},
	)
	receipt, err := svc.Import(
		ctx,
		&bankreceiptports.ImportBankReceiptRequest{
			ReceiptDate:     now.Unix(),
			AmountMinor:     10000,
			ReferenceNumber: "BANK-MATCH-1",
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.Equal(t, bankreceipt.StatusImported, receipt.Status)

	matched, err := svc.Match(
		ctx,
		&bankreceiptports.MatchBankReceiptRequest{
			ReceiptID: receipt.ID,
			PaymentID: payment.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.Equal(t, bankreceipt.StatusMatched, matched.Status)
	assert.Equal(t, payment.ID, matched.MatchedCustomerPaymentID)
	require.NotNil(t, matched.MatchedAt)
	assert.Equal(t, user.ID, matched.MatchedByID)
}

func int64Ptr(v int64) *int64 { return &v }
