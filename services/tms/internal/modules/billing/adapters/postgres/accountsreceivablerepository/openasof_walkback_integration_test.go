//go:build integration

package accountsreceivablerepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const walkBackWeeks = 13

type asOfChange struct {
	at    int64
	delta int64
}

type asOfHistory struct {
	id          pulid.ID
	receivable  bool
	invoiceDate int64
	totalMinor  int64
	changes     []asOfChange
}

func (h *asOfHistory) settle(at, amountMinor int64, releasedAt *int64) {
	h.changes = append(h.changes, asOfChange{at: at, delta: -amountMinor})
	if releasedAt != nil {
		h.changes = append(h.changes, asOfChange{at: max(at, *releasedAt), delta: amountMinor})
	}
}

func (h *asOfHistory) appliedMinor() int64 {
	var applied int64
	for _, change := range h.changes {
		applied -= change.delta
	}
	return applied
}

func (h *asOfHistory) openAt(at int64) int64 {
	var open int64
	if h.invoiceDate <= at {
		open = h.totalMinor
	}
	for _, change := range h.changes {
		if change.at <= at {
			open += change.delta
		}
	}
	return open
}

type walkBackFixture struct {
	t         *testing.T
	ctx       context.Context
	db        *keel.DB
	tenant    *billingtest.Tenant
	invoices  invoiceports.InvoiceRepository
	histories []*asOfHistory
	byID      map[pulid.ID]*asOfHistory
	line      int
}

func (f *walkBackFixture) fullSum(asOf int64, weeks int) []int64 {
	totals := make([]int64, weeks)
	for idx := range weeks {
		periodEnd := asOf - int64(weeks-1-idx)*secondsPerWeek
		for _, history := range f.histories {
			if !history.receivable || history.invoiceDate > asOf {
				continue
			}
			if open := history.openAt(periodEnd); open > 0 {
				totals[idx] += open
			}
		}
	}
	return totals
}

func (f *walkBackFixture) post(
	number string,
	billType billingqueue.BillType,
	total, invoiceDate int64,
	dueDate *int64,
) *invoice.Invoice {
	t := f.t
	org, shp := f.tenant.Org, f.tenant.Shipment
	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		ShipmentID:       shp.ID,
		BillToCustomerID: shp.CustomerID,
		Number:           number,
		Status:           billingqueue.StatusPosted,
		BillType:         billType,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(f.ctx, f.db, queue))
	postedAt := invoiceDate
	amount := decimal.NewFromInt(total)
	inv, err := f.invoices.Create(f.ctx, &invoice.Invoice{
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
		BillToName:         "Walk Back Customer",
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
	history := &asOfHistory{
		id:          inv.ID,
		receivable:  billType != billingqueue.BillTypeCreditMemo,
		invoiceDate: invoiceDate,
		totalMinor:  total * 100,
	}
	f.histories = append(f.histories, history)
	f.byID[inv.ID] = history
	return inv
}

func (f *walkBackFixture) backdate(table string, id pulid.ID, createdAt int64) {
	_, err := keel.Exec(
		f.ctx,
		f.db,
		"UPDATE "+table+" SET created_at = $1 WHERE id = $2 AND organization_id = $3 AND business_unit_id = $4",
		createdAt,
		id.String(),
		f.tenant.Org.ID.String(),
		f.tenant.Org.BusinessUnitID.String(),
	)
	require.NoError(f.t, err)
}

func (f *walkBackFixture) pay(
	number string,
	inv *invoice.Invoice,
	appliedMinor, shortPayMinor, appliedAt int64,
	reversedAt *int64,
) {
	status := customerpayment.StatusPosted
	if reversedAt != nil {
		status = customerpayment.StatusReversed
	}
	payment := &customerpayment.Payment{
		OrganizationID:     f.tenant.Org.ID,
		BusinessUnitID:     f.tenant.Org.BusinessUnitID,
		CustomerID:         f.tenant.Shipment.CustomerID,
		PaymentDate:        appliedAt,
		AccountingDate:     appliedAt,
		AmountMinor:        appliedMinor,
		AppliedAmountMinor: appliedMinor,
		Status:             status,
		PaymentMethod:      customerpayment.MethodACH,
		ReferenceNumber:    number,
		CurrencyCode:       "USD",
		ReversedAt:         reversedAt,
		CreatedByID:        f.tenant.User.ID,
	}
	require.NoError(f.t, customerpaymentdb.Payments.Create(f.ctx, f.db, payment))
	application := &customerpayment.Application{
		OrganizationID:      f.tenant.Org.ID,
		BusinessUnitID:      f.tenant.Org.BusinessUnitID,
		CustomerPaymentID:   payment.ID,
		InvoiceID:           inv.ID,
		AppliedAmountMinor:  appliedMinor,
		ShortPayAmountMinor: shortPayMinor,
		LineNumber:          1,
	}
	require.NoError(f.t, customerpaymentdb.Applications.Create(f.ctx, f.db, application))
	f.backdate("customer_payment_applications", application.ID, appliedAt)
	f.byID[inv.ID].settle(appliedAt, appliedMinor+shortPayMinor, reversedAt)
}

func (f *walkBackFixture) credit(
	memo, inv *invoice.Invoice,
	amountMinor, appliedAt int64,
	unappliedAt *int64,
) {
	f.line++
	application := &customerpayment.CreditMemoApplication{
		OrganizationID:      f.tenant.Org.ID,
		BusinessUnitID:      f.tenant.Org.BusinessUnitID,
		CreditMemoInvoiceID: memo.ID,
		InvoiceID:           inv.ID,
		AppliedAmountMinor:  amountMinor,
		AccountingDate:      appliedAt,
		LineNumber:          f.line,
		Status:              customerpayment.CreditApplicationStatusApplied,
		CreatedByID:         f.tenant.User.ID,
	}
	if unappliedAt != nil {
		application.Status = customerpayment.CreditApplicationStatusUnapplied
		application.UnappliedAt = unappliedAt
		application.UnappliedByID = f.tenant.User.ID
	}
	require.NoError(f.t, customerpaymentdb.CreditMemoApplications.Create(f.ctx, f.db, application))
	f.backdate("credit_memo_applications", application.ID, appliedAt)
	f.byID[inv.ID].settle(appliedAt, amountMinor, unappliedAt)
}

func (f *walkBackFixture) factor(inv *invoice.Invoice, sourceID string, ledgerMinor, at int64) {
	entry := &customerledger.CustomerLedgerEntry{
		OrganizationID:   f.tenant.Org.ID,
		BusinessUnitID:   f.tenant.Org.BusinessUnitID,
		CustomerID:       f.tenant.Shipment.CustomerID,
		SourceObjectType: customerledger.SourceFactoringItem,
		SourceObjectID:   sourceID,
		SourceEventType:  accountingcontrol.JournalSourceEventFactoringFunded.String(),
		RelatedInvoiceID: inv.ID,
		DocumentNumber:   inv.Number,
		TransactionDate:  at,
		LineNumber:       1,
		AmountMinor:      ledgerMinor,
		CreatedByID:      f.tenant.User.ID,
	}
	require.NoError(f.t, customerledgerdb.CustomerLedgerEntries.Create(f.ctx, f.db, entry))
	f.backdate("customer_ledger_entries", entry.ID, at)
	history := f.byID[inv.ID]
	history.changes = append(history.changes, asOfChange{at: at, delta: ledgerMinor})
}

func (f *walkBackFixture) void(inv *invoice.Invoice, at int64) {
	invs := invoicedb.Invoices
	_, err := invs.Where(invs.ID.Eq(inv.ID)).Update(
		f.ctx,
		f.db,
		invs.Status.To(invoice.StatusVoided),
		invs.VoidedAt.To(at),
	)
	require.NoError(f.t, err)
	f.byID[inv.ID].receivable = false
}

func (f *walkBackFixture) storeApplied() {
	invs := invoicedb.Invoices
	for _, history := range f.histories {
		applied := history.appliedMinor()
		_, err := invs.Where(invs.ID.Eq(history.id)).Update(
			f.ctx,
			f.db,
			invs.AppliedAmountMinor.To(applied),
			invs.AppliedAmount.To(decimal.New(applied, -2)),
		)
		require.NoError(f.t, err)
	}
}

type walkBackRead struct {
	balances  []int64
	agingOpen []int64
	beginning int64
}

func TestOpenAsOfWalkBackMatchesFullSum(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	repo := NewAnalytics(Params{DB: conn, Logger: logger})

	seeded := billingtest.SeededTenant(t, ctx, db)
	f := &walkBackFixture{
		t:        t,
		ctx:      seeded.Ctx,
		db:       db,
		tenant:   seeded,
		invoices: invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger}),
		byID:     make(map[pulid.ID]*asOfHistory),
	}
	tenantInfo := pagination.TenantInfo{OrgID: seeded.Org.ID, BuID: seeded.Org.BusinessUnitID}

	now := time.Now().UTC().Unix()
	exactly := func(weeksAgo int64) int64 { return now - weeksAgo*secondsPerWeek }
	ago := func(weeksAgo int64) int64 { return exactly(weeksAgo) - 3600 }
	at := func(v int64) *int64 { return &v }

	asOfs := []int64{now, exactly(3)}
	read := func(asOf int64) walkBackRead {
		series := customerledgerports.ListARSeriesRequest{
			TenantInfo: tenantInfo,
			AsOfDate:   asOf,
			Weeks:      walkBackWeeks,
		}
		balances, err := repo.ListBalanceSeries(seeded.Ctx, series)
		require.NoError(t, err)
		require.Len(t, balances, walkBackWeeks)
		aging, err := repo.ListAgingTrend(seeded.Ctx, series)
		require.NoError(t, err)
		require.Len(t, aging, walkBackWeeks)
		totals, err := repo.GetCollectionTotals(
			seeded.Ctx,
			customerledgerports.GetARCollectionMetricsRequest{
				TenantInfo: tenantInfo,
				AsOfDate:   asOf,
				PeriodDays: customerledger.CollectionWindowDays,
			},
		)
		require.NoError(t, err)

		out := walkBackRead{beginning: totals.BeginningOpenMinor}
		for idx := range walkBackWeeks {
			assert.Equal(t, asOf-int64(walkBackWeeks-1-idx)*secondsPerWeek, balances[idx].PeriodEnd)
			out.balances = append(out.balances, balances[idx].ARBalanceMinor)
			out.agingOpen = append(out.agingOpen, aging[idx].Buckets.TotalOpenMinor)
		}
		return out
	}

	before := make(map[int64]walkBackRead, len(asOfs))
	for _, asOf := range asOfs {
		before[asOf] = read(asOf)
	}

	memo := f.post("CM-WB-MEMO", billingqueue.BillTypeCreditMemo, 500, ago(30), nil)

	across := f.post("INV-WB-ACROSS", billingqueue.BillTypeInvoice, 1000, ago(20), at(ago(16)))
	f.pay("PAY-WB-ACROSS-1", across, 30_000, 0, ago(18), nil)
	f.pay("PAY-WB-ACROSS-2", across, 20_000, 0, ago(10), nil)
	f.pay("PAY-WB-ACROSS-SHORT", across, 15_000, 5_000, ago(4), nil)

	reversedIn := f.post("INV-WB-REVERSED-IN", billingqueue.BillTypeInvoice, 800, ago(9), at(ago(5)))
	f.pay("PAY-WB-REVERSED-IN", reversedIn, 80_000, 0, ago(7), at(ago(2)))
	f.credit(memo, reversedIn, 10_000, ago(1), nil)

	reversedAcross := f.post(
		"INV-WB-REVERSED-ACROSS",
		billingqueue.BillTypeInvoice,
		600,
		ago(25),
		at(ago(21)),
	)
	f.pay("PAY-WB-REVERSED-ACROSS", reversedAcross, 60_000, 0, ago(13), at(ago(6)))
	f.pay("PAY-WB-REPAID-ACROSS", reversedAcross, 60_000, 0, ago(1), nil)

	oldReversal := f.post("INV-WB-OLD-REVERSAL", billingqueue.BillTypeInvoice, 500, ago(30), at(ago(26)))
	f.pay("PAY-WB-OLD-REVERSAL", oldReversal, 50_000, 0, ago(28), at(ago(26)))

	credited := f.post("INV-WB-CREDIT", billingqueue.BillTypeInvoice, 400, ago(16), nil)
	f.credit(memo, credited, 15_000, ago(14), at(ago(8)))
	f.credit(memo, credited, 5_000, exactly(5), at(ago(2)))
	f.credit(memo, credited, 10_000, ago(11), nil)

	factored := f.post("INV-WB-FACTORED", billingqueue.BillTypeInvoice, 900, ago(11), at(ago(7)))
	f.factor(factored, "fsi_wb_funded", -90_000, ago(6))
	f.factor(factored, "fsi_wb_recourse", 40_000, ago(2))

	boundary := f.post("INV-WB-BOUNDARY", billingqueue.BillTypeInvoice, 300, exactly(12), nil)
	f.pay("PAY-WB-BOUNDARY", boundary, 30_000, 0, exactly(12), nil)

	boundaryReversal := f.post(
		"INV-WB-BOUNDARY-REVERSAL",
		billingqueue.BillTypeInvoice,
		200,
		ago(15),
		at(ago(11)),
	)
	f.pay("PAY-WB-BOUNDARY-REVERSED", boundaryReversal, 20_000, 0, ago(14), at(exactly(12)))
	f.pay("PAY-WB-BOUNDARY-REPAID", boundaryReversal, 20_000, 0, exactly(9), nil)

	voided := f.post("INV-WB-VOIDED", billingqueue.BillTypeInvoice, 700, ago(8), at(ago(4)))
	f.pay("PAY-WB-VOIDED", voided, 70_000, 0, ago(6), at(ago(4)))
	f.void(voided, ago(3))

	debit := f.post("DM-WB-NEW", billingqueue.BillTypeDebitMemo, 250, ago(2), at(exactly(-2)))
	f.pay("PAY-WB-DEBIT", debit, 5_000, 0, ago(1), nil)

	reversedLate := f.post("INV-WB-REVERSED-LATE", billingqueue.BillTypeInvoice, 350, ago(20), at(ago(16)))
	f.pay("PAY-WB-REVERSED-LATE", reversedLate, 35_000, 0, ago(16), at(ago(1)))

	f.storeApplied()

	for _, asOf := range asOfs {
		after := read(asOf)
		was := before[asOf]
		expected := f.fullSum(asOf, walkBackWeeks)
		var moved int64
		for idx := range walkBackWeeks {
			moved += expected[idx]
			assert.Equal(
				t,
				expected[idx],
				after.balances[idx]-was.balances[idx],
				"balance as of %d, point %d",
				asOf,
				idx,
			)
			assert.Equal(
				t,
				expected[idx],
				after.agingOpen[idx]-was.agingOpen[idx],
				"aging total as of %d, point %d",
				asOf,
				idx,
			)
		}
		assert.Positive(t, moved, "the fixtures leave an open balance in the window as of %d", asOf)

		periodStart := asOf - int64(customerledger.CollectionWindowDays)*secondsPerDay
		assert.Equal(
			t,
			f.fullSum(periodStart, 1)[0],
			after.beginning-was.beginning,
			"beginning open as of %d",
			asOf,
		)
	}
}
