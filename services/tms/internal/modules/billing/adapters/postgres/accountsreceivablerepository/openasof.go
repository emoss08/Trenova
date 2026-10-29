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

const (
	colPeriodEnd  = "period_end"
	colInvoiceID  = "invoice_id"
	colDueDate    = "due_date"
	colSettledAt  = "settled_at"
	colReleasedAt = "released_at"
	colAmount     = "amount"
	colChangedAt  = "changed_at"
	colDelta      = "delta"
	colOpenNow    = "open_now_minor"
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
	series := keel.CTE("series", []string{"n"}, keel.Rows(
		"series_n",
		[]string{"n"},
		keel.GenerateSeries(keel.Int64(0), keel.Int64(int64(w.points-1))),
	))
	seriesN := keel.Int64Of(series.Col("n"))
	points := keel.CTE("points", []string{"n", colPeriodEnd}, series.Query().Pick(
		seriesN,
		keel.Int64(w.asOf).Sub(seriesN.Mul(keel.Int64(w.step))),
	))

	events := w.settlementEvents()
	candidates := w.candidates(events)
	moves := w.movesOf(candidates, events)
	changes := w.changesOf(candidates, moves)
	segments := segmentsOf(changes)
	items := w.openItems(segments)

	return openAsOf{
		ctes: []*keel.CTETable{
			series,
			points,
			events,
			candidates,
			moves,
			changes,
			segments,
			items,
		},
		points: points,
		items:  items,
	}
}

func (w openAsOfWindow) settlementEvents() *keel.CTETable {
	start := w.start()
	cpa := customerpaymentdb.Applications.As("cpa")
	cp := customerpaymentdb.Payments.As("cp")
	reversed := customerpaymentdb.Payments.As("cp")
	reversedApp := customerpaymentdb.Applications.As("cpa")
	cma := customerpaymentdb.CreditMemoApplications.As("cma")
	unapplied := customerpaymentdb.CreditMemoApplications.As("cma")
	cle := customerledgerdb.CustomerLedgerEntries.As("cle")

	return keel.CTE(
		"events",
		[]string{colInvoiceID, colSettledAt, colReleasedAt, colAmount},
		keel.UnionAll(
			cpa.
				Join(cp, cp.ID.EqCol(cpa.CustomerPaymentID)).
				Pick(
					cpa.InvoiceID,
					cpa.CreatedAt,
					cp.ReversedAt,
					cpa.AppliedAmountMinor.Add(cpa.ShortPayAmountMinor),
				).
				Where(cpa.CreatedAt.Gt(start)),
			reversed.
				Join(reversedApp, reversedApp.CustomerPaymentID.EqCol(reversed.ID)).
				Pick(
					reversedApp.InvoiceID,
					reversedApp.CreatedAt,
					reversed.ReversedAt,
					reversedApp.AppliedAmountMinor.Add(reversedApp.ShortPayAmountMinor),
				).
				Where(reversed.ReversedAt.Gt(start), reversedApp.CreatedAt.Lte(start)),
			cma.
				Pick(cma.InvoiceID, cma.CreatedAt, cma.UnappliedAt, cma.AppliedAmountMinor).
				Where(cma.CreatedAt.Gt(start)),
			unapplied.
				Pick(
					unapplied.InvoiceID,
					unapplied.CreatedAt,
					unapplied.UnappliedAt,
					unapplied.AppliedAmountMinor,
				).
				Where(unapplied.UnappliedAt.Gt(start), unapplied.CreatedAt.Lte(start)),
			cle.
				Pick(cle.RelatedInvoiceID, cle.CreatedAt, keel.NullInt64(), cle.AmountMinor.Neg()).
				Where(
					cle.SourceObjectType.EqCol(keel.ConstText(customerledger.SourceFactoringItem)),
					cle.RelatedInvoiceID.IsNotNull(),
					cle.CreatedAt.Gt(start),
				),
		),
	)
}

func (w openAsOfWindow) candidates(events *keel.CTETable) *keel.CTETable {
	open := invoicedb.Invoices.As("inv")
	touched := invoicedb.Invoices.As("inv")
	return keel.CTE(
		"candidates",
		[]string{"id", "invoice_date", colDueDate, "total_amount_minor", colOpenNow},
		keel.Union(
			open.
				Pick(
					open.ID,
					open.InvoiceDate,
					open.DueDate,
					open.TotalAmountMinor,
					receivablesql.OpenMinor(open),
				).
				Where(open.InvoiceDate.Lte(w.asOf), receivablesql.Open(open)),
			touched.
				Join(events, events.Col(colInvoiceID).EqCol(touched.ID)).
				Pick(
					touched.ID,
					touched.InvoiceDate,
					touched.DueDate,
					touched.TotalAmountMinor,
					receivablesql.OpenMinor(touched),
				).
				Where(receivablesql.Receivable(touched), touched.InvoiceDate.Lte(w.asOf)),
		),
	)
}

func (w openAsOfWindow) movesOf(candidates, events *keel.CTETable) *keel.CTETable {
	start := w.start()
	id := candidates.Col("id")
	due := candidates.Col(colDueDate)
	invoiceDate := candidates.Col("invoice_date")
	settledAt := keel.Int64Of(events.Col(colSettledAt))
	releasedAt := events.Col(colReleasedAt)
	releasedOn := settledAt.Greatest(keel.Int64Of(releasedAt))
	amount := keel.Int64Of(events.Col(colAmount))
	onCandidate := func() keel.Pred { return id.EqCol(events.Col(colInvoiceID)) }

	return keel.CTE(
		"moves",
		[]string{colInvoiceID, colDueDate, colChangedAt, colDelta},
		keel.UnionAll(
			candidates.Query().
				Pick(id, due, invoiceDate, candidates.Col("total_amount_minor")).
				Where(invoiceDate.Gt(start)),
			events.Query().
				Join(candidates, onCandidate()).
				Pick(id, due, settledAt, amount.Neg()).
				Where(settledAt.Gt(start)),
			events.Query().
				Join(candidates, onCandidate()).
				Pick(id, due, releasedOn, amount).
				Where(releasedAt.IsNotNull(), releasedOn.Gt(start)),
		),
	)
}

func (w openAsOfWindow) changesOf(candidates, moves *keel.CTETable) *keel.CTETable {
	start := keel.Int64(w.start())

	return keel.CTE(
		"changes",
		[]string{colInvoiceID, colDueDate, colChangedAt, colDelta},
		keel.UnionAll(
			candidates.Query().Pick(
				candidates.Col("id"),
				candidates.Col(colDueDate),
				start,
				candidates.Col(colOpenNow),
			),
			moves.Query().Pick(
				moves.Col(colInvoiceID),
				moves.Col(colDueDate),
				moves.Col(colChangedAt),
				moves.Col(colDelta),
			),
			moves.Query().Pick(
				moves.Col(colInvoiceID),
				moves.Col(colDueDate),
				start,
				keel.Int64Of(moves.Col(colDelta)).Neg(),
			),
		),
	)
}

func segmentsOf(changes *keel.CTETable) *keel.CTETable {
	id := changes.Col(colInvoiceID)
	due := changes.Col(colDueDate)
	changedAt := changes.Col(colChangedAt)
	window := keel.NamedWindow("w")

	return keel.CTE(
		"segments",
		[]string{colInvoiceID, colDueDate, "starts_at", "ends_at", "open_minor"},
		changes.Query().
			Pick(
				id,
				due,
				changedAt,
				keel.Lead(changedAt, 1, nil).OverWindow(window),
				keel.Sum(keel.Sum(changes.Col(colDelta))).OverWindow(window),
			).
			Group(id, due, changedAt).
			Window("w", keel.PartitionBy(id).OrderBy(changedAt.Asc())),
	)
}

func (w openAsOfWindow) openItems(segments *keel.CTETable) *keel.CTETable {
	startsAt := keel.Int64Of(segments.Col("starts_at"))
	endsAt := keel.Int64Of(segments.Col("ends_at"))
	periodsBefore := func(at keel.Int64Value) keel.Int64Expr {
		return keel.CastInt64(
			keel.CastNumeric(keel.Int64(w.asOf).Sub(at)).Div(keel.Numeric(w.step)).Floor(),
		)
	}
	firstOffset := keel.CaseInt64().
		When(endsAt.IsNull(), keel.Int64(0)).
		Else(keel.Int64(0).Greatest(periodsBefore(endsAt).Add(keel.Int64(1))))
	offsets := keel.Rows("offsets", []string{"n"}, keel.GenerateSeries(
		firstOffset,
		keel.Int64(int64(w.points-1)).Least(periodsBefore(startsAt)),
	))

	return keel.CTE(
		"open_items",
		[]string{"n", colInvoiceID, colDueDate, "open_minor"},
		segments.Query().
			JoinLateral(offsets).
			Pick(
				offsets.Col("n"),
				segments.Col(colInvoiceID),
				segments.Col(colDueDate),
				segments.Col("open_minor"),
			).
			Where(segments.Col("open_minor").Gt(0)),
	)
}
