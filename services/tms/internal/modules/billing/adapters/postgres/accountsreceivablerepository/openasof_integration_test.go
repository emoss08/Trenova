//go:build integration

package accountsreceivablerepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/billingtest"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const asOfTestWeeks = 13

func TestOpenAsOfFollowsEverySettlementSource(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	repo := NewAnalytics(Params{DB: conn, Logger: logger})
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx, org, user, shp := seeded.Ctx, seeded.Org, seeded.User, seeded.Shipment

	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	asOf := time.Now().UTC().Unix()
	before := func(weeksAgo int64) int64 { return asOf - weeksAgo*secondsPerWeek - 3600 }

	series := func() []*customerledgerports.ARBalancePoint {
		points, sErr := repo.ListBalanceSeries(ctx, customerledgerports.ListARSeriesRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			Weeks:      asOfTestWeeks,
		})
		require.NoError(t, sErr)
		require.Len(t, points, asOfTestWeeks)
		return points
	}
	aging := func() []*customerledgerports.ARAgingTrendPoint {
		points, aErr := repo.ListAgingTrend(ctx, customerledgerports.ListARSeriesRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			Weeks:      asOfTestWeeks,
		})
		require.NoError(t, aErr)
		require.Len(t, points, asOfTestWeeks)
		return points
	}

	seriesBefore := series()
	agingBefore := aging()

	post := func(
		number string,
		billType billingqueue.BillType,
		total int64,
		invoiceDate int64,
		dueDate *int64,
	) *invoice.Invoice {
		queue := &billingqueue.BillingQueueItem{
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			ShipmentID:       shp.ID,
			BillToCustomerID: shp.CustomerID,
			Number:           number,
			Status:           billingqueue.StatusPosted,
			BillType:         billType,
		}
		qErr := billingqueuedb.BillingQueueItems.Create(ctx, db, queue)
		require.NoError(t, qErr)
		postedAt := invoiceDate
		amount := decimal.NewFromInt(total)
		inv, iErr := invoiceRepo.Create(ctx, &invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shp.CustomerID,
			Number:             number,
			BillType:           billType,
			Status:             invoice.StatusPosted,
			PostedAt:           &postedAt,
			PaymentTerm:        invoice.PaymentTermNet30,
			CurrencyCode:       "USD",
			InvoiceDate:        invoiceDate,
			DueDate:            dueDate,
			ShipmentProNumber:  shp.ProNumber,
			ShipmentBOL:        shp.BOL,
			BillToName:         "As Of Customer",
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
		require.NoError(t, iErr)
		return inv
	}
	settle := func(inv *invoice.Invoice, appliedMinor int64) {
		invs := invoicedb.Invoices
		_, uErr := invs.Where(invs.ID.Eq(inv.ID)).Update(
			ctx,
			db,
			invs.AppliedAmountMinor.To(appliedMinor),
			invs.AppliedAmount.To(decimal.New(appliedMinor, -2)),
		)
		require.NoError(t, uErr)
	}
	backdate := func(table string, id pulid.ID, createdAt int64) {
		_, uErr := keel.Exec(
			ctx,
			db,
			"UPDATE "+table+" SET created_at = $1 WHERE id = $2 AND organization_id = $3 AND business_unit_id = $4",
			createdAt,
			id.String(),
			org.ID.String(),
			org.BusinessUnitID.String(),
		)
		require.NoError(t, uErr)
	}
	pay := func(
		number string,
		inv *invoice.Invoice,
		amountMinor int64,
		appliedAt int64,
		reversedAt *int64,
	) {
		status := customerpayment.StatusPosted
		if reversedAt != nil {
			status = customerpayment.StatusReversed
		}
		payment := &customerpayment.Payment{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			CustomerID:         shp.CustomerID,
			PaymentDate:        appliedAt,
			AccountingDate:     appliedAt,
			AmountMinor:        amountMinor,
			AppliedAmountMinor: amountMinor,
			Status:             status,
			PaymentMethod:      customerpayment.MethodACH,
			ReferenceNumber:    number,
			CurrencyCode:       "USD",
			ReversedAt:         reversedAt,
			CreatedByID:        user.ID,
		}
		pErr := customerpaymentdb.Payments.Create(ctx, db, payment)
		require.NoError(t, pErr)
		application := &customerpayment.Application{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			CustomerPaymentID:  payment.ID,
			InvoiceID:          inv.ID,
			AppliedAmountMinor: amountMinor,
			LineNumber:         1,
		}
		aErr := customerpaymentdb.Applications.Create(ctx, db, application)
		require.NoError(t, aErr)
		backdate("customer_payment_applications", application.ID, appliedAt)
	}

	factored := post("INV-ASOF-FACTORED", billingqueue.BillTypeInvoice, 1000, before(10),
		int64PtrAR(asOf-6*secondsPerWeek))
	memo := post("CM-ASOF-FACTORED", billingqueue.BillTypeCreditMemo, 100, before(11), nil)
	reopened := post("DM-ASOF-REOPENED", billingqueue.BillTypeDebitMemo, 500, before(3),
		int64PtrAR(asOf+4*secondsPerWeek))

	pay("PAY-ASOF-PARTIAL", factored, 40_000, before(8), nil)
	credit := &customerpayment.CreditMemoApplication{
		OrganizationID:      org.ID,
		BusinessUnitID:      org.BusinessUnitID,
		CreditMemoInvoiceID: memo.ID,
		InvoiceID:           factored.ID,
		AppliedAmountMinor:  10_000,
		AccountingDate:      before(6),
		LineNumber:          1,
		Status:              customerpayment.CreditApplicationStatusApplied,
		CreatedByID:         user.ID,
	}
	err := customerpaymentdb.CreditMemoApplications.Create(ctx, db, credit)
	require.NoError(t, err)
	backdate("credit_memo_applications", credit.ID, before(6))
	factoring := &customerledger.CustomerLedgerEntry{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		CustomerID:       shp.CustomerID,
		SourceObjectType: customerledger.SourceFactoringItem,
		SourceObjectID:   "fsi_asof_settlement",
		SourceEventType:  tenant.JournalSourceEventFactoringFunded.String(),
		RelatedInvoiceID: factored.ID,
		DocumentNumber:   factored.Number,
		TransactionDate:  before(4),
		LineNumber:       1,
		AmountMinor:      -50_000,
		CreatedByID:      user.ID,
	}
	err = customerledgerdb.CustomerLedgerEntries.Create(ctx, db, factoring)
	require.NoError(t, err)
	backdate("customer_ledger_entries", factoring.ID, before(4))
	settle(factored, 100_000)

	reversedAt := before(1)
	pay("PAY-ASOF-REVERSED", reopened, 50_000, before(2), &reversedAt)

	seriesAfter := series()
	agingAfter := aging()

	expectedByWeeksAgo := map[int]int64{
		10: 100_000,
		9:  100_000,
		8:  60_000,
		7:  60_000,
		6:  50_000,
		5:  50_000,
		3:  50_000,
		1:  50_000,
		0:  50_000,
	}
	for idx := range asOfTestWeeks {
		weeksAgo := asOfTestWeeks - 1 - idx
		assert.Equal(t, asOf-int64(weeksAgo)*secondsPerWeek, seriesAfter[idx].PeriodEnd)
		assert.Equal(
			t,
			expectedByWeeksAgo[weeksAgo],
			seriesAfter[idx].ARBalanceMinor-seriesBefore[idx].ARBalanceMinor,
			"balance moved %d weeks ago",
			weeksAgo,
		)
		assert.Equal(
			t,
			expectedByWeeksAgo[weeksAgo],
			agingAfter[idx].Buckets.TotalOpenMinor-agingBefore[idx].Buckets.TotalOpenMinor,
			"aging total %d weeks ago",
			weeksAgo,
		)
	}

	pastDue := asOfTestWeeks - 1 - 5
	assert.Equal(
		t,
		int64(50_000),
		agingAfter[pastDue].Buckets.Days1To30Minor-agingBefore[pastDue].Buckets.Days1To30Minor,
	)
	notYetDue := asOfTestWeeks - 1 - 8
	assert.Equal(
		t,
		int64(60_000),
		agingAfter[notYetDue].Buckets.CurrentMinor-agingBefore[notYetDue].Buckets.CurrentMinor,
	)

	overview, err := repo.GetBalanceOverview(
		ctx,
		customerledgerports.GetARAnalyticsRequest{TenantInfo: tenantInfo, AsOfDate: asOf},
	)
	require.NoError(t, err)
	assert.True(t, overview.HasInvoiced)
	today := seriesAfter[asOfTestWeeks-1]
	assert.Equal(t, overview.TotalOpenMinor, today.ARBalanceMinor)
	assert.Equal(t, overview.Buckets, agingAfter[asOfTestWeeks-1].Buckets)

	totals, err := repo.GetCollectionTotals(
		ctx,
		customerledgerports.GetARCollectionMetricsRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			PeriodDays: customerledger.CollectionWindowDays,
		},
	)
	require.NoError(t, err)
	opening, err := repo.ListBalanceSeries(ctx, customerledgerports.ListARSeriesRequest{
		TenantInfo: tenantInfo,
		AsOfDate:   totals.PeriodStart,
		Weeks:      1,
	})
	require.NoError(t, err)
	require.Len(t, opening, 1)
	assert.Equal(t, opening[0].ARBalanceMinor, totals.BeginningOpenMinor)
}
