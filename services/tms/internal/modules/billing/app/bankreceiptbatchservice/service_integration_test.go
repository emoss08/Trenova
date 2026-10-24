//go:build integration

package bankreceiptbatchservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch/bankreceiptbatchdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptbatchrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptworkitemrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/customerpaymentservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
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

type batchSeededOrg = billingtest.Org

type batchSeededUser = billingtest.User

type batchSeededShipment = billingtest.Shipment

type batchIntegrationEnv struct {
	ctx          context.Context
	db           *keel.DB
	batchService *Service
	org          batchSeededOrg
	user         batchSeededUser
	shipment     batchSeededShipment
	now          time.Time
	cleanup      func()
}

func TestImportBatchPersistsMixedOutcomesAndSupportsReads(t *testing.T) {
	env := setupBatchIntegrationEnv(t)
	defer env.cleanup()

	createPostedCustomerPayment(t, env, "BANK-MATCH-1", 10000)

	actor := testutil.NewSessionActor(env.user.ID, env.org.ID, env.org.BusinessUnitID)
	result, err := env.batchService.Import(env.ctx, &bankreceiptbatchports.ImportBankReceiptBatchRequest{
		Source:    "csv",
		Reference: "BATCH-MIXED-1",
		Receipts: []*bankreceiptbatchports.ImportBankReceiptBatchLine{
			{
				ReceiptDate:     env.now.Unix(),
				AmountMinor:     10000,
				ReferenceNumber: "BANK-MATCH-1",
				Memo:            "auto-match",
			},
			{
				ReceiptDate:     env.now.Add(time.Hour).Unix(),
				AmountMinor:     5000,
				ReferenceNumber: "NO-MATCH-1",
				Memo:            "exception",
			},
		},
		TenantInfo: pagination.TenantInfo{
			OrgID:  env.org.ID,
			BuID:   env.org.BusinessUnitID,
			UserID: env.user.ID,
		},
	}, actor)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Batch)
	assert.Equal(t, int64(2), result.Batch.ImportedCount)
	assert.Equal(t, int64(15000), result.Batch.ImportedAmountMinor)
	assert.Equal(t, int64(1), result.Batch.MatchedCount)
	assert.Equal(t, int64(10000), result.Batch.MatchedAmountMinor)
	assert.Equal(t, int64(1), result.Batch.ExceptionCount)
	assert.Equal(t, int64(5000), result.Batch.ExceptionAmountMinor)
	require.Len(t, result.Receipts, 2)
	for _, receipt := range result.Receipts {
		assert.Equal(t, result.Batch.ID, receipt.ImportBatchID)
	}

	detail, err := env.batchService.Get(
		env.ctx,
		&bankreceiptbatchports.GetBankReceiptBatchRequest{
			BatchID: result.Batch.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  env.org.ID,
				BuID:   env.org.BusinessUnitID,
				UserID: env.user.ID,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.NotNil(t, detail.Batch)
	assert.Equal(t, result.Batch.ID, detail.Batch.ID)
	require.Len(t, detail.Receipts, 2)
	assert.Equal(t, bankreceipt.StatusMatched, detail.Receipts[0].Status)
	assert.Equal(t, bankreceipt.StatusException, detail.Receipts[1].Status)
	for _, receipt := range detail.Receipts {
		assert.Equal(t, result.Batch.ID, receipt.ImportBatchID)
	}

	batches, err := env.batchService.List(
		env.ctx,
		pagination.TenantInfo{OrgID: env.org.ID, BuID: env.org.BusinessUnitID, UserID: env.user.ID},
	)
	require.NoError(t, err)
	require.NotEmpty(t, batches)
	assert.Equal(t, result.Batch.ID, batches[0].ID)

	br := bankreceiptdb.BankReceipts
	count, err := br.Where(br.ImportBatchID.Eq(result.Batch.ID)).Count(env.ctx, env.db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestImportBatchRollsBackOnNilReceiptLine(t *testing.T) {
	env := setupBatchIntegrationEnv(t)
	defer env.cleanup()

	createPostedCustomerPayment(t, env, "ROLLBACK-MATCH-1", 10000)

	actor := testutil.NewSessionActor(env.user.ID, env.org.ID, env.org.BusinessUnitID)
	result, err := env.batchService.Import(env.ctx, &bankreceiptbatchports.ImportBankReceiptBatchRequest{
		Source:    "csv",
		Reference: "BATCH-ROLLBACK-1",
		Receipts: []*bankreceiptbatchports.ImportBankReceiptBatchLine{
			{
				ReceiptDate:     env.now.Unix(),
				AmountMinor:     10000,
				ReferenceNumber: "ROLLBACK-MATCH-1",
				Memo:            "should rollback",
			},
			nil,
		},
		TenantInfo: pagination.TenantInfo{
			OrgID:  env.org.ID,
			BuID:   env.org.BusinessUnitID,
			UserID: env.user.ID,
		},
	}, actor)

	require.Error(t, err)
	assert.Nil(t, result)

	batches := bankreceiptbatchdb.BankReceiptBatches
	batchCount, err := batches.Where(batches.Reference.Eq("BATCH-ROLLBACK-1")).Count(env.ctx, env.db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), batchCount)

	receipts := bankreceiptdb.BankReceipts
	receiptCount, err := receipts.Where(receipts.ReferenceNumber.Eq("ROLLBACK-MATCH-1")).
		Count(env.ctx, env.db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), receiptCount)
}

func setupBatchIntegrationEnv(t *testing.T) *batchIntegrationEnv {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	receiptRepo := bankreceiptrepository.New(bankreceiptrepository.Params{DB: conn, Logger: logger})
	batchRepo := bankreceiptbatchrepository.New(
		bankreceiptbatchrepository.Params{DB: conn, Logger: logger},
	)
	workItemRepo := bankreceiptworkitemrepository.New(
		bankreceiptworkitemrepository.Params{DB: conn, Logger: logger},
	)
	paymentRepo := customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger},
	)
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx
	org, user, shipment := seeded.Org, seeded.User, seeded.Shipment

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = billingtest.GLAccountID(t, ctx, db, "1010")
	control.DefaultUnappliedCashAccountID = billingtest.GLAccountID(t, ctx, db, "2200")
	control.DefaultARAccountID = billingtest.GLAccountID(t, ctx, db, "1110")
	control.ReconciliationMode = tenant.ReconciliationModeWarnOnly
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	receiptSvc := bankreceiptservice.New(
		bankreceiptservice.Params{
			Logger:         logger,
			DB:             conn,
			Repo:           receiptRepo,
			WorkItemRepo:   workItemRepo,
			PaymentRepo:    paymentRepo,
			AccountingRepo: accountingRepo,
			AuditService:   &mocks.NoopAuditService{},
		},
	)
	batchSvc := New(
		Params{
			Logger:             logger,
			DB:                 conn,
			Repo:               batchRepo,
			ReceiptRepo:        receiptRepo,
			BankReceiptService: receiptSvc,
			AuditService:       &mocks.NoopAuditService{},
		},
	)

	return &batchIntegrationEnv{
		ctx:          ctx,
		db:           db,
		batchService: batchSvc,
		org:          org,
		user:         user,
		shipment:     shipment,
		now:          time.Now().UTC(),
		cleanup:      cleanup,
	}
}

func createPostedCustomerPayment(
	t *testing.T,
	env *batchIntegrationEnv,
	referenceNumber string,
	amountMinor int64,
) {
	t.Helper()

	conn := postgres.NewTestConnection(env.db)
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

	now := env.now
	_, period := fiscaltest.OpenPeriod(t, env.ctx, fiscaltest.OpenPeriodParams{
		DB:             env.db,
		OrganizationID: env.org.ID,
		BusinessUnitID: env.org.BusinessUnitID,
		At:             now.Unix(),
	})

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   env.org.ID,
		BusinessUnitID:   env.org.BusinessUnitID,
		ShipmentID:       env.shipment.ID,
		BillToCustomerID: env.shipment.CustomerID,
		Number:           "INV-" + referenceNumber,
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	err := billingqueuedb.BillingQueueItems.Create(env.ctx, env.db, queue)
	require.NoError(t, err)
	postedAt := now.Unix()
	inv, err := invoiceRepo.Create(
		env.ctx,
		&invoice.Invoice{
			OrganizationID:     env.org.ID,
			BusinessUnitID:     env.org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         env.shipment.ID,
			CustomerID:         env.shipment.CustomerID,
			Number:             queue.Number,
			BillType:           billingqueue.BillTypeInvoice,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
			PaymentTerm:        invoice.PaymentTermNet30,
			CurrencyCode:       "USD",
			InvoiceDate:        period.StartDate,
			DueDate:            int64Ptr(period.EndDate),
			ShipmentProNumber:  env.shipment.ProNumber,
			ShipmentBOL:        env.shipment.BOL,
			BillToName:         "Test Customer",
			SubtotalAmount:     decimal.NewFromInt(amountMinor / 100),
			OtherAmount:        decimal.Zero,
			TotalAmount:        decimal.NewFromInt(amountMinor / 100),
			AppliedAmount:      decimal.Zero,
			SettlementStatus:   invoice.SettlementStatusUnpaid,
			DisputeStatus:      invoice.DisputeStatusNone,
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber:  1,
					Type:        invoice.InvoiceLineTypeFreight,
					Description: "Freight",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(amountMinor / 100),
					Amount:      decimal.NewFromInt(amountMinor / 100),
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
	_, err = paymentSvc.PostAndApply(
		env.ctx,
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:      env.shipment.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     amountMinor,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: referenceNumber,
			CurrencyCode:    "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: inv.ID, AppliedAmountMinor: amountMinor},
			},
			TenantInfo: pagination.TenantInfo{
				OrgID:  env.org.ID,
				BuID:   env.org.BusinessUnitID,
				UserID: env.user.ID,
			},
		},
		testutil.NewSessionActor(env.user.ID, env.org.ID, env.org.BusinessUnitID),
	)
	require.NoError(t, err)
}

func int64Ptr(v int64) *int64 { return &v }
