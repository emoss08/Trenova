//go:build integration

package collectionrepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerledgerrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/factoringcompanyrepository"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type nonRecourseEnv struct {
	ctx        context.Context
	db         *bun.DB
	tenantInfo pagination.TenantInfo
	customerID pulid.ID
	shipment   seededShipment
	invoices   invoiceports.InvoiceRepository
	ledger     customerledgerports.CustomerLedgerProjectionRepository
	factors    factoringcompanyports.FactoringCompanyRepository
	stateID    pulid.ID
	dueDate    int64
}

type seededShipment struct {
	ID         pulid.ID `bun:"id"`
	CustomerID pulid.ID `bun:"customer_id"`
	ProNumber  string   `bun:"pro_number"`
	BOL        string   `bun:"bol"`
}

func TestNonRecourseFactoredInvoicesLeaveCollectionsAndCreditExposure(t *testing.T) {
	env := setupNonRecourseEnv(t)
	repo := New(Params{DB: postgres.NewTestConnection(env.db), Logger: zap.NewNop()})

	recourse := env.createFactor(t, "RECRS", false)
	nonRecourse := env.createFactor(t, "NONREC", true)

	plain := env.postInvoice(t, "INV-NR-1", 10000, nil)
	withRecourse := env.postInvoice(t, "INV-NR-2", 20000, &recourse.ID)
	sold := env.postInvoice(t, "INV-NR-3", 40000, &nonRecourse.ID)
	env.appendPayment(t, sold, 5000)

	row, err := repo.GetWorklistRow(env.ctx, collectionports.GetAccountRequest{
		TenantInfo: env.tenantInfo,
		CustomerID: env.customerID,
	}, time.Now().Unix())
	require.NoError(t, err)
	assert.Equal(t, plain.TotalAmountMinor+withRecourse.TotalAmountMinor, row.OverdueMinor)
	assert.Equal(t, plain.TotalAmountMinor+withRecourse.TotalAmountMinor, row.TotalOpenMinor)
	assert.Equal(t, 2, row.OpenInvoiceCount)

	all, err := env.ledger.SumCustomerBalances(env.ctx, customerledgerports.SumCustomerBalancesRequest{
		TenantInfo:  env.tenantInfo,
		CustomerIDs: []pulid.ID{env.customerID},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(10000+20000+40000-5000), all[env.customerID])

	exposure, err := env.ledger.SumCustomerBalances(
		env.ctx,
		customerledgerports.SumCustomerBalancesRequest{
			TenantInfo:                 env.tenantInfo,
			CustomerIDs:                []pulid.ID{env.customerID},
			ExcludeNonRecourseFactored: true,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(10000+20000), exposure[env.customerID])

	nonRecourse.NonRecourse = false
	_, err = env.factors.Update(env.ctx, nonRecourse)
	require.NoError(t, err)

	row, err = repo.GetWorklistRow(env.ctx, collectionports.GetAccountRequest{
		TenantInfo: env.tenantInfo,
		CustomerID: env.customerID,
	}, time.Now().Unix())
	require.NoError(t, err)
	assert.Equal(t, 3, row.OpenInvoiceCount)
}

func TestStampedAgreementDecidesRecourse(t *testing.T) {
	env := setupNonRecourseEnv(t)
	repo := New(Params{DB: postgres.NewTestConnection(env.db), Logger: zap.NewNop()})
	agreements := factoringrepository.NewAgreementRepository(factoringrepository.Params{
		DB:     postgres.NewTestConnection(env.db),
		Logger: zap.NewNop(),
	})

	flaggedSold := env.createFactor(t, "FLAGSOLD", true)
	flaggedRecourse := env.createFactor(t, "FLAGRCRS", false)
	now := time.Now().Unix()
	withRecourse := env.agreeWith(t, agreements, flaggedSold.ID, factoring.RecourseFull, now)
	withoutRecourse := env.agreeWith(t, agreements, flaggedRecourse.ID, factoring.RecourseNone, now)

	kept := env.postInvoice(t, "INV-AGR-1", 10000, &flaggedSold.ID)
	sold := env.postInvoice(t, "INV-AGR-2", 20000, &flaggedRecourse.ID)
	env.stampAgreement(t, kept, withRecourse.ID)
	env.stampAgreement(t, sold, withoutRecourse.ID)
	env.postInvoice(t, "INV-AGR-3", 40000, &flaggedSold.ID)

	row, err := repo.GetWorklistRow(env.ctx, collectionports.GetAccountRequest{
		TenantInfo: env.tenantInfo,
		CustomerID: env.customerID,
	}, time.Now().Unix())
	require.NoError(t, err)
	assert.Equal(t, 1, row.OpenInvoiceCount, "only the invoice under a recourse agreement stays")
	assert.Equal(t, kept.TotalAmountMinor, row.TotalOpenMinor)

	exposure, err := env.ledger.SumCustomerBalances(
		env.ctx,
		customerledgerports.SumCustomerBalancesRequest{
			TenantInfo:                 env.tenantInfo,
			CustomerIDs:                []pulid.ID{env.customerID},
			ExcludeNonRecourseFactored: true,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, kept.TotalAmountMinor, exposure[env.customerID])
}

func (e *nonRecourseEnv) agreeWith(
	t *testing.T,
	repo factoringports.AgreementRepository,
	factorID pulid.ID,
	recourse factoring.Recourse,
	now int64,
) *factoring.Agreement {
	t.Helper()
	created, err := repo.Create(e.ctx, &factoring.Agreement{
		OrganizationID:       e.tenantInfo.OrgID,
		BusinessUnitID:       e.tenantInfo.BuID,
		FactoringCompanyID:   factorID,
		Status:               factoring.AgreementStatusActive,
		ActivatedAt:          new(now - 86400),
		EffectiveFrom:        now - 86400,
		AdvanceRateBP:        9000,
		ReserveRateBP:        1000,
		ChargebackWindowDays: 90,
		Recourse:             recourse,
		AccountingTreatment:  factoring.AccountingTreatmentSale,
		FundingDays:          1,
		SubmissionTime:       600,
		FeeTiers:             []*factoring.FeeTier{{DaysFrom: 0, RateBP: 300}},
	})
	require.NoError(t, err)
	return created
}

func (e *nonRecourseEnv) stampAgreement(t *testing.T, inv *invoice.Invoice, agreementID pulid.ID) {
	t.Helper()
	_, err := e.db.NewUpdate().
		Table("invoices").
		Set("factoring_agreement_id = ?", agreementID).
		Where("id = ?", inv.ID).
		Where("organization_id = ?", e.tenantInfo.OrgID).
		Where("business_unit_id = ?", e.tenantInfo.BuID).
		Exec(e.ctx)
	require.NoError(t, err)
}

func TestFactoringCompanyUpdateSavesClearedFields(t *testing.T) {
	env := setupNonRecourseEnv(t)

	factor := env.createFactor(t, "CLEAR", true)
	require.NotEmpty(t, factor.Email)
	createdAt := factor.CreatedAt

	factor.Email = ""
	factor.Notes = ""
	factor.NonRecourse = false
	_, err := env.factors.Update(env.ctx, factor)
	require.NoError(t, err)

	saved, err := env.factors.GetByID(env.ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:         factor.ID,
		TenantInfo: env.tenantInfo,
	})
	require.NoError(t, err)
	assert.Empty(t, saved.Email)
	assert.Empty(t, saved.Notes)
	assert.False(t, saved.NonRecourse)
	assert.Equal(t, createdAt, saved.CreatedAt)
}

func setupNonRecourseEnv(t *testing.T) *nonRecourseEnv {
	t.Helper()
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))

	var user struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(t, db.NewSelect().
		Table("users").
		Column("id").
		Where("current_organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Limit(1).
		Scan(ctx, &user))

	var shp seededShipment
	require.NoError(t, db.NewSelect().
		Table("shipments").
		Column("id", "customer_id", "pro_number", "bol").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Apply(seedtest.ShipmentsWithoutReceivables).
		Limit(1).
		Scan(ctx, &shp))

	var stateID pulid.ID
	require.NoError(t, db.NewSelect().
		Table("us_states").
		Column("id").
		Limit(1).
		Scan(ctx, &stateID))

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	return &nonRecourseEnv{
		ctx: ctx,
		db:  db,
		tenantInfo: pagination.TenantInfo{
			OrgID:  org.ID,
			BuID:   org.BusinessUnitID,
			UserID: user.ID,
		},
		customerID: shp.CustomerID,
		shipment:   shp,
		invoices:   invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger}),
		ledger: customerledgerrepository.New(
			customerledgerrepository.Params{DB: conn, Logger: logger},
		),
		factors: factoringcompanyrepository.New(
			factoringcompanyrepository.Params{DB: conn, Logger: logger},
		),
		stateID: stateID,
		dueDate: time.Now().AddDate(0, 0, -20).Unix(),
	}
}

func (e *nonRecourseEnv) createFactor(
	t *testing.T,
	code string,
	nonRecourse bool,
) *factoringcompany.FactoringCompany {
	t.Helper()
	created, err := e.factors.Create(e.ctx, &factoringcompany.FactoringCompany{
		OrganizationID: e.tenantInfo.OrgID,
		BusinessUnitID: e.tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Code:           code,
		Name:           code + " Capital",
		AddressLine1:   "100 Main St",
		City:           "Dallas",
		StateID:        e.stateID,
		PostalCode:     "75201",
		Email:          "remit@" + code + ".example.com",
		Notes:          "Agreement on file",
		NonRecourse:    nonRecourse,
	})
	require.NoError(t, err)
	return created
}

func (e *nonRecourseEnv) postInvoice(
	t *testing.T,
	number string,
	amountMinor int64,
	factorID *pulid.ID,
) *invoice.Invoice {
	t.Helper()
	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   e.tenantInfo.OrgID,
		BusinessUnitID:   e.tenantInfo.BuID,
		ShipmentID:       e.shipment.ID,
		BillToCustomerID: e.customerID,
		Number:           number,
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
	}
	_, err := e.db.NewInsert().Model(queue).Exec(e.ctx)
	require.NoError(t, err)

	amount := decimal.New(amountMinor, -2)
	postedAt := time.Now().Unix()
	dueDate := e.dueDate
	inv, err := e.invoices.Create(e.ctx, &invoice.Invoice{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         e.shipment.ID,
		CustomerID:         e.customerID,
		FactoringCompanyID: factorID,
		Number:             number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		PostedAt:           &postedAt,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        dueDate - 30*86400,
		DueDate:            &dueDate,
		ShipmentProNumber:  e.shipment.ProNumber,
		ShipmentBOL:        e.shipment.BOL,
		BillToName:         "Test Customer",
		SubtotalAmount:     amount,
		OtherAmount:        decimal.Zero,
		TotalAmount:        amount,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Freight",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   amount,
			Amount:      amount,
		}},
	})
	require.NoError(t, err)

	require.NoError(t, e.ledger.AppendEntries(e.ctx, []*customerledger.CustomerLedgerEntry{{
		ID:               pulid.MustNew("cledg_"),
		OrganizationID:   e.tenantInfo.OrgID,
		BusinessUnitID:   e.tenantInfo.BuID,
		CustomerID:       e.customerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   inv.ID.String(),
		SourceEventType:  tenant.JournalSourceEventInvoicePosted.String(),
		DocumentNumber:   inv.Number,
		TransactionDate:  postedAt,
		LineNumber:       1,
		AmountMinor:      inv.TotalAmountMinor,
		CreatedByID:      e.tenantInfo.UserID,
	}}))
	return inv
}

func (e *nonRecourseEnv) appendPayment(t *testing.T, inv *invoice.Invoice, amountMinor int64) {
	t.Helper()
	require.NoError(t, e.ledger.AppendEntries(e.ctx, []*customerledger.CustomerLedgerEntry{{
		ID:               pulid.MustNew("cledg_"),
		OrganizationID:   e.tenantInfo.OrgID,
		BusinessUnitID:   e.tenantInfo.BuID,
		CustomerID:       e.customerID,
		SourceObjectType: "CustomerPayment",
		SourceObjectID:   pulid.MustNew("cpay_").String(),
		SourceEventType:  tenant.JournalSourceEventCustomerPaymentPosted.String(),
		RelatedInvoiceID: inv.ID,
		DocumentNumber:   "PAY-" + inv.Number,
		TransactionDate:  time.Now().Unix(),
		LineNumber:       1,
		AmountMinor:      -amountMinor,
		CreatedByID:      e.tenantInfo.UserID,
	}}))
}
