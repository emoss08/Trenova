package accountsreceivablerepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
)

type openAsOfWindow struct {
	asOf   int64
	step   int64
	points int
}

func seriesWindow(req customerledgerports.ListARSeriesRequest) openAsOfWindow {
	return openAsOfWindow{asOf: req.AsOfDate, step: secondsPerWeek, points: req.Weeks}
}

func instantWindow(asOf int64) openAsOfWindow {
	return openAsOfWindow{asOf: asOf, step: secondsPerWeek, points: 1}
}

func (w openAsOfWindow) start() int64 {
	return w.asOf - int64(w.points-1)*w.step
}

type openAsOf struct {
	ctes   []*keel.CTETable
	points *keel.CTETable
	items  *keel.CTETable
}

func (w openAsOfWindow) build() openAsOf {
	series := keel.CTE("series", []string{"n"}, keel.Rows("series_n", []string{"n"}, keel.DynFunc(
		"generate_series",
		keel.DynLit(int64(0)).Cast(keel.TypeInt64),
		keel.DynLit(int64(w.points-1)).Cast(keel.TypeInt64),
	)))
	seriesN := keel.DynOf(series.Col("n"))
	points := keel.CTE("points", []string{"n", "period_end"}, series.Query().Pick(
		seriesN,
		keel.DynLit(w.asOf).Cast(keel.TypeInt64).Sub(seriesN.Mul(w.step)),
	))

	moved := w.movedInvoices()
	candidates := w.candidates(moved)
	settlements := settlementsOf(candidates)
	changes := changesOf(candidates, settlements)
	segments := segmentsOf(changes)
	items := w.openItems(segments)

	return openAsOf{
		ctes: []*keel.CTETable{
			series,
			points,
			moved,
			candidates,
			settlements,
			changes,
			segments,
			items,
		},
		points: points,
		items:  items,
	}
}

func (w openAsOfWindow) movedInvoices() *keel.CTETable {
	start := w.start()
	applied := customerpaymentdb.Applications.As("cpa")
	reversedApp := customerpaymentdb.Applications.As("cpa")
	reversed := customerpaymentdb.Payments.As("cp")
	memo := customerpaymentdb.CreditMemoApplications.As("cma")
	ledger := customerledgerdb.CustomerLedgerEntries.As("cle")

	return keel.CTE("moved", []string{"invoice_id"}, keel.Union(
		applied.Pick(applied.InvoiceID).Where(applied.CreatedAt.Gt(start)),
		reversed.
			Join(reversedApp, reversedApp.CustomerPaymentID.EqCol(reversed.ID)).
			Pick(reversedApp.InvoiceID).
			Where(reversed.ReversedAt.Gt(start)),
		memo.Pick(memo.InvoiceID).Where(memo.CreatedAt.Gt(start)),
		memo.Pick(memo.InvoiceID).Where(memo.UnappliedAt.Gt(start)),
		ledger.
			Pick(ledger.RelatedInvoiceID).
			Where(
				ledger.SourceObjectType.Eq(customerledger.SourceFactoringItem),
				ledger.RelatedInvoiceID.IsNotNull(),
				ledger.CreatedAt.Gt(start),
			),
	))
}

func (w openAsOfWindow) candidates(moved *keel.CTETable) *keel.CTETable {
	open := invoicedb.Invoices.As("inv")
	touched := invoicedb.Invoices.As("inv")
	return keel.CTE(
		"candidates",
		[]string{"id", "invoice_date", "due_date", "total_amount_minor"},
		keel.Union(
			open.
				Pick(open.ID, open.InvoiceDate, open.DueDate, open.TotalAmountMinor).
				Where(open.InvoiceDate.Lte(w.asOf), receivablesql.Open(open)),
			touched.
				Join(moved, moved.Col("invoice_id").EqCol(touched.ID)).
				Pick(touched.ID, touched.InvoiceDate, touched.DueDate, touched.TotalAmountMinor).
				Where(receivablesql.Receivable(touched), touched.InvoiceDate.Lte(w.asOf)),
		),
	)
}

func settlementsOf(candidates *keel.CTETable) *keel.CTETable {
	id := candidates.Col("id")
	due := candidates.Col("due_date")
	cpa := customerpaymentdb.Applications.As("cpa")
	cp := customerpaymentdb.Payments.As("cp")
	cma := customerpaymentdb.CreditMemoApplications.As("cma")
	cle := customerledgerdb.CustomerLedgerEntries.As("cle")

	return keel.CTE(
		"settlements",
		[]string{"invoice_id", "due_date", "settled_at", "released_at", "amount"},
		keel.UnionAll(
			cpa.
				Join(candidates, id.EqCol(cpa.InvoiceID)).
				Join(cp, cp.ID.EqCol(cpa.CustomerPaymentID)).
				Pick(
					id,
					due,
					cpa.CreatedAt,
					cp.ReversedAt,
					keel.DynOf(cpa.AppliedAmountMinor).Add(cpa.ShortPayAmountMinor),
				),
			cma.
				Join(candidates, id.EqCol(cma.InvoiceID)).
				Pick(id, due, cma.CreatedAt, cma.UnappliedAt, cma.AppliedAmountMinor),
			cle.
				Join(
					candidates,
					id.EqCol(cle.RelatedInvoiceID),
					cle.SourceObjectType.Eq(customerledger.SourceFactoringItem),
				).
				Pick(id, due, cle.CreatedAt, keel.DynLit(nil), keel.DynOf(cle.AmountMinor).Neg()),
		),
	)
}

func changesOf(candidates, settlements *keel.CTETable) *keel.CTETable {
	settledAt := keel.DynOf(settlements.Col("settled_at"))
	releasedAt := keel.DynOf(settlements.Col("released_at"))
	amount := keel.DynOf(settlements.Col("amount"))

	return keel.CTE(
		"changes",
		[]string{"invoice_id", "due_date", "changed_at", "delta"},
		keel.UnionAll(
			candidates.Query().Pick(
				candidates.Col("id"),
				candidates.Col("due_date"),
				candidates.Col("invoice_date"),
				candidates.Col("total_amount_minor"),
			),
			settlements.Query().Pick(
				settlements.Col("invoice_id"),
				settlements.Col("due_date"),
				settledAt,
				amount.Neg(),
			),
			settlements.Query().
				Pick(
					settlements.Col("invoice_id"),
					settlements.Col("due_date"),
					keel.DynFunc("greatest", settledAt, releasedAt),
					amount,
				).
				Where(releasedAt.IsNotNull()),
		),
	)
}

func segmentsOf(changes *keel.CTETable) *keel.CTETable {
	id := changes.Col("invoice_id")
	due := changes.Col("due_date")
	changedAt := changes.Col("changed_at")
	window := keel.NamedWindow("w")

	return keel.CTE(
		"segments",
		[]string{"invoice_id", "due_date", "starts_at", "ends_at", "open_minor"},
		changes.Query().
			Pick(
				id,
				due,
				changedAt,
				keel.Lead(changedAt, 1, nil).OverWindow(window),
				keel.Sum(keel.Sum(changes.Col("delta"))).OverWindow(window),
			).
			Group(id, due, changedAt).
			Window("w", keel.PartitionBy(id).OrderBy(changedAt.Asc())),
	)
}

func (w openAsOfWindow) openItems(segments *keel.CTETable) *keel.CTETable {
	startsAt := keel.DynOf(segments.Col("starts_at"))
	endsAt := keel.DynOf(segments.Col("ends_at"))
	periodsBefore := func(at keel.DynExpr) keel.DynExpr {
		return keel.DynFunc(
			"floor",
			keel.DynLit(w.asOf).Sub(at).Cast(keel.TypeNumeric).Div(w.step),
		).Cast(keel.TypeInt64)
	}
	offsets := keel.Rows("offsets", []string{"n"}, keel.DynFunc(
		"generate_series",
		keel.DynCase().
			When(endsAt.IsNull(), 0).
			Else(keel.DynFunc("greatest", 0, periodsBefore(endsAt).Add(1))),
		keel.DynFunc("least", int64(w.points-1), periodsBefore(startsAt)),
	))

	return keel.CTE(
		"open_items",
		[]string{"n", "invoice_id", "due_date", "open_minor"},
		segments.Query().
			JoinLateral(offsets).
			Pick(
				offsets.Col("n"),
				segments.Col("invoice_id"),
				segments.Col("due_date"),
				segments.Col("open_minor"),
			).
			Where(keel.DynOf(segments.Col("open_minor")).Gt(0)),
	)
}
