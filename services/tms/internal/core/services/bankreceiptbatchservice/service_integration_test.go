//go:build integration

package bankreceiptbatchservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/internal/core/services/customerpaymentservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/bankreceiptbatchrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/bankreceiptrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/bankreceiptworkitemrepository"
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

type batchSeededOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type batchSeededUser struct {
	ID pulid.ID `bun:"id"`
}

type batchSeededShipment struct {
	ID         pulid.ID `bun:"id"`
	CustomerID pulid.ID `bun:"customer_id"`
	ProNumber  string   `bun:"pro_number"`
	BOL        string   `bun:"bol"`
}

type batchIntegrationEnv struct {
	ctx          context.Context
	db           *bun.DB
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
	result, err := env.batchService.Import(env.ctx, &serviceports.ImportBankReceiptBatchRequest{
		Source:    "csv",
		Reference: "BATCH-MIXED-1",
		Receipts: []*serviceports.ImportBankReceiptBatchLine{
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
		&serviceports.GetBankReceiptBatchRequest{
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

	count, err := env.db.NewSelect().
		Table("bank_receipts").
		Where("import_batch_id = ?", result.Batch.ID).
		Where("organization_id = ?", env.org.ID).
		Where("business_unit_id = ?", env.org.BusinessUnitID).
		Count(env.ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestImportBatchRollsBackOnNilReceiptLine(t *testing.T) {
	env := setupBatchIntegrationEnv(t)
	defer env.cleanup()

	createPostedCustomerPayment(t, env, "ROLLBACK-MATCH-1", 10000)

	actor := testutil.NewSessionActor(env.user.ID, env.org.ID, env.org.BusinessUnitID)
	result, err := env.batchService.Import(env.ctx, &serviceports.ImportBankReceiptBatchRequest{
		Source:    "csv",
		Reference: "BATCH-ROLLBACK-1",
		Receipts: []*serviceports.ImportBankReceiptBatchLine{
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

	batchCount, err := env.db.NewSelect().
		Table("bank_receipt_import_batches").
		Where("reference = ?", "BATCH-ROLLBACK-1").
		Where("organization_id = ?", env.org.ID).
		Where("business_unit_id = ?", env.org.BusinessUnitID).
		Count(env.ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, batchCount)

	receiptCount, err := env.db.NewSelect().
		Table("bank_receipts").
		Where("reference_number = ?", "ROLLBACK-MATCH-1").
		Where("organization_id = ?", env.org.ID).
		Where("business_unit_id = ?", env.org.BusinessUnitID).
		Count(env.ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, receiptCount)
}

func setupBatchIntegrationEnv(t *testing.T) *batchIntegrationEnv {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
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

	var org batchSeededOrg
	require.NoError(
		t,
		db.NewSelect().
			Table("organizations").
			Column("id", "business_unit_id").
			Limit(1).
			Scan(ctx, &org),
	)
	var user batchSeededUser
	require.NoError(
		t,
		db.NewSelect().
			Table("users").
			Column("id").
			Where("current_organization_id = ?", org.ID).
			Where("business_unit_id = ?", org.BusinessUnitID).
			Limit(1).
			Scan(ctx, &user),
	)
	var shipment batchSeededShipment
	require.NoError(
		t,
		db.NewSelect().
			Table("shipments").
			Column("id", "customer_id", "pro_number", "bol").
			Where("organization_id = ?", org.ID).
			Where("business_unit_id = ?", org.BusinessUnitID).
			Limit(1).
			Scan(ctx, &shipment),
	)

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.DefaultCashAccountID = lookupBatchGLAccount(
		t,
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		"1010",
	)
	control.DefaultUnappliedCashAccountID = lookupBatchGLAccount(
		t,
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		"2200",
	)
	control.DefaultARAccountID = lookupBatchGLAccount(
		t,
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		"1110",
	)
	control.ReconciliationMode = tenant.ReconciliationModeWarnOnly
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	receiptSvc := bankreceiptservice.New(
		bankreceiptservice.Params{
			Logger:         logger,
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
	fiscalYearRepo := fiscalyearrepository.New(
		fiscalyearrepository.Params{DB: conn, Logger: logger},
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
	fy, err := fiscalYearRepo.Create(
		env.ctx,
		&fiscalyear.FiscalYear{
			OrganizationID:        env.org.ID,
			BusinessUnitID:        env.org.BusinessUnitID,
			Status:                fiscalyear.StatusOpen,
			Year:                  now.Year(),
			Name:                  "FY",
			StartDate:             now.Add(-24 * time.Hour).Unix(),
			EndDate:               now.Add(24 * time.Hour).Unix(),
			IsCurrent:             true,
			AllowAdjustingEntries: true,
		},
	)
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(
		env.ctx,
		&fiscalperiod.FiscalPeriod{
			OrganizationID:        env.org.ID,
			BusinessUnitID:        env.org.BusinessUnitID,
			FiscalYearID:          fy.ID,
			PeriodNumber:          1,
			PeriodType:            fiscalperiod.PeriodTypeMonth,
			Status:                fiscalperiod.StatusOpen,
			Name:                  "Period",
			StartDate:             now.Add(-24 * time.Hour).Unix(),
			EndDate:               now.Add(24 * time.Hour).Unix(),
			AllowAdjustingEntries: true,
		},
	)
	require.NoError(t, err)

	queue := &billingqueue.BillingQueueItem{
		OrganizationID: env.org.ID,
		BusinessUnitID: env.org.BusinessUnitID,
		ShipmentID:     env.shipment.ID,
		Number:         "INV-" + referenceNumber,
		Status:         billingqueue.StatusPosted,
		BillType:       billingqueue.BillTypeInvoice,
	}
	_, err = env.db.NewInsert().Model(queue).Exec(env.ctx)
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
			Lines: []*invoice.InoviceLine{
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
		&serviceports.PostCustomerPaymentRequest{
			CustomerID:      env.shipment.CustomerID,
			PaymentDate:     now.Unix(),
			AccountingDate:  now.Unix(),
			AmountMinor:     amountMinor,
			PaymentMethod:   customerpayment.MethodACH,
			ReferenceNumber: referenceNumber,
			CurrencyCode:    "USD",
			Applications: []*serviceports.CustomerPaymentApplicationInput{
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

func lookupBatchGLAccount(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	orgID, buID pulid.ID,
	accountCode string,
) pulid.ID {
	t.Helper()

	var row struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(
		t,
		db.NewSelect().
			Table("gl_accounts").
			Column("id").
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Where("account_code = ?", accountCode).
			Limit(1).
			Scan(ctx, &row),
	)
	return row.ID
}

func int64Ptr(v int64) *int64 { return &v }
