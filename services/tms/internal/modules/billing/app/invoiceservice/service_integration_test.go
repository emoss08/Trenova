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
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededInvoiceOrg struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

type seededInvoiceUser struct {
	ID pulid.ID
}

type seededInvoiceShipment struct {
	ID         pulid.ID
	CustomerID pulid.ID
	ProNumber  string
	BOL        string
}

func TestPostCreatesInvoiceJournalSourceAndBalances(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	billingQueueRepo := billingqueuerepository.New(
		billingqueuerepository.Params{DB: conn, Logger: logger},
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
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	ctx, org, user, shp := seededInvoiceFixture(t, ctx, db)

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.ReconciliationMode = accountingcontrol.ReconciliationModeDisabled
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
		t,
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		"4000",
	)
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
		Number:           "INV-1001",
		Status:           billingqueue.StatusApproved,
		BillType:         billingqueue.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, queue))

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
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Freight",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(100),
			Amount:      decimal.NewFromInt(100),
		}},
	})
	require.NoError(t, err)

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req != nil && req.ID == shp.ID && req.TenantInfo.OrgID == org.ID &&
				req.TenantInfo.BuID == org.BusinessUnitID
		})).
		Return(&shipment.Shipment{
			ID:             shp.ID,
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Status:         shipment.StatusReadyToInvoice,
		}, nil)
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
			return entity != nil && entity.ID == shp.ID &&
				entity.Status == shipment.StatusInvoiced &&
				entity.BilledAt != nil
		})).
		Return(&shipment.Shipment{ID: shp.ID}, nil)

	svc := &Service{
		l:                logger,
		db:               conn,
		repo:             invoiceRepo,
		billingQueueRepo: billingQueueRepo,
		shipmentRepo:     shipmentRepo,
		accountingRepo:   accountingRepo,
		journalRepo:      journalRepo,
		validator: NewValidator(
			ValidatorParams{
				DB:               conn,
				Logger:           logger,
				AccountingRepo:   accountingRepo,
				FiscalPeriodRepo: fiscalPeriodRepo,
				ShipmentRepo:     shipmentRepo,
			},
		),
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		sequenceGenerator: generator,
	}

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
	revenueBefore := balanceOf(control.DefaultRevenueAccountID)

	posted, err := svc.Post(
		ctx,
		&invoiceports.PostInvoiceRequest{
			InvoiceID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.Equal(t, invoice.StatusPosted, posted.Status)
	require.NotNil(t, posted.PostedAt)

	var entry struct {
		ReferenceType string
		ReferenceID   string
		TotalDebit    int64
		TotalCredit   int64
	}
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT reference_type, reference_id, total_debit, total_credit
		FROM journal_entries WHERE reference_id = $1 LIMIT 1`,
		entity.ID.String(),
	).Scan(&entry.ReferenceType, &entry.ReferenceID, &entry.TotalDebit, &entry.TotalCredit))
	assert.Equal(t, accountingcontrol.JournalSourceEventInvoicePosted.String(), entry.ReferenceType)
	assert.Equal(t, entity.ID.String(), entry.ReferenceID)
	assert.Equal(t, int64(10000), entry.TotalDebit)
	assert.Equal(t, int64(10000), entry.TotalCredit)

	var source struct {
		SourceEventType string
		Status          string
	}
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT source_event_type, status FROM journal_sources WHERE source_object_id = $1 LIMIT 1",
		entity.ID.String(),
	).Scan(&source.SourceEventType, &source.Status))
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventInvoicePosted.String(),
		source.SourceEventType,
	)
	assert.Equal(t, "Posted", source.Status)

	arAfter := balanceOf(control.DefaultARAccountID)
	assert.Equal(t, int64(10000), arAfter.PeriodDebitMinor-arBefore.PeriodDebitMinor)

	revenueAfter := balanceOf(control.DefaultRevenueAccountID)
	assert.Equal(
		t,
		int64(10000),
		revenueAfter.PeriodCreditMinor-revenueBefore.PeriodCreditMinor,
	)
}

func TestPostSkipsInvoiceLedgerWhenRevenueRecognitionIsCashReceipt(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	billingQueueRepo := billingqueuerepository.New(
		billingqueuerepository.Params{DB: conn, Logger: logger},
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
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	ctx, org, user, shp := seededInvoiceFixture(t, ctx, db)

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.AccountingBasis = accountingcontrol.AccountingBasisCash
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnCashReceipt
	control.ReconciliationMode = accountingcontrol.ReconciliationModeDisabled
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.AutoPostSourceEvents = []accountingcontrol.JournalSourceEventType{
		accountingcontrol.JournalSourceEventCustomerPaymentPosted,
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
		t,
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		"4000",
	)
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
		Number:           "INV-1002",
		Status:           billingqueue.StatusApproved,
		BillType:         billingqueue.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, queue))
	entity, err := invoiceRepo.Create(
		ctx,
		&invoice.Invoice{
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

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shp.ID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: shipment.StatusReadyToInvoice}, nil)
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shp.ID}, nil)

	svc := &Service{
		l:                logger,
		db:               conn,
		repo:             invoiceRepo,
		billingQueueRepo: billingQueueRepo,
		shipmentRepo:     shipmentRepo,
		accountingRepo:   accountingRepo,
		journalRepo:      journalRepo,
		validator: NewValidator(
			ValidatorParams{
				DB:               conn,
				Logger:           logger,
				AccountingRepo:   accountingRepo,
				FiscalPeriodRepo: fiscalPeriodRepo,
				ShipmentRepo:     shipmentRepo,
			},
		),
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		sequenceGenerator: generator,
	}
	posted, err := svc.Post(
		ctx,
		&invoiceports.PostInvoiceRequest{
			InvoiceID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  org.ID,
				BuID:   org.BusinessUnitID,
				UserID: user.ID,
			},
		},
		testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID),
	)
	require.NoError(t, err)
	require.Equal(t, invoice.StatusPosted, posted.Status)

	var entryCount, sourceCount int64
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT count(*) FROM journal_entries WHERE reference_id = $1", entity.ID.String(),
	).Scan(&entryCount))
	assert.Equal(t, int64(0), entryCount)
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT count(*) FROM journal_sources WHERE source_object_id = $1", entity.ID.String(),
	).Scan(&sourceCount))
	assert.Equal(t, int64(0), sourceCount)
}

func lookupInvoiceGLAccount(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
	accountCode string,
) pulid.ID {
	t.Helper()

	var id pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id FROM gl_accounts WHERE organization_id = $1 AND business_unit_id = $2 AND account_code = $3 LIMIT 1",
		orgID, buID, accountCode,
	).
		Scan(&id))
	return id
}

func seededInvoiceFixture(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
) (context.Context, seededInvoiceOrg, seededInvoiceUser, seededInvoiceShipment) {
	t.Helper()

	var org seededInvoiceOrg
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&org.ID, &org.BusinessUnitID))

	var user seededInvoiceUser
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id FROM users WHERE current_organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
		org.ID, org.BusinessUnitID,
	).
		Scan(&user.ID))

	var shp seededInvoiceShipment
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT id, customer_id, COALESCE(pro_number, ''), COALESCE(bol, '') FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1`,
		org.ID, org.BusinessUnitID,
	).Scan(&shp.ID, &shp.CustomerID, &shp.ProNumber, &shp.BOL))

	info := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}
	return dbscope.WithTenant(ctx, info.DBTenant()), org, user, shp
}
