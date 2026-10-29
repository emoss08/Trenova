package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringsql"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	secondsPerDay      = int64(86400)
	secondsPerWeek     = int64(604800)
	trailingDSOWindow  = 91 * secondsPerDay
	trailingYearWindow = 365 * secondsPerDay
)

func inDays(seconds keel.Int64Value) keel.NumericExpr {
	return keel.CastNumeric(seconds).Div(keel.Numeric(secondsPerDay))
}

func orZero(e keel.Expr) keel.NumericExpr {
	return keel.NumericOf(e).Coalesce(0)
}

func billedIn(inv *invoicedb.InvoicesTable, from, to keel.Int64Value) keel.Pred {
	return keel.And(
		inv.PostedAt.IsNotNull(),
		keel.Or(inv.VoidedAt.IsNull(), keel.Int64Of(inv.VoidedAt).GtExpr(to)),
		receivablesql.ReceivableBillType(inv),
		keel.Int64Of(inv.InvoiceDate).GtExpr(from),
		keel.Int64Of(inv.InvoiceDate).LteExpr(to),
	)
}

func avgDaysToPay(
	cpa *customerpaymentdb.ApplicationsTable,
	cp *customerpaymentdb.PaymentsTable,
	inv *invoicedb.InvoicesTable,
	preds ...keel.Pred,
) keel.NumericExpr {
	weighted := keel.CastNumeric(cpa.AppliedAmountMinor).
		Mul(inDays(cp.PaymentDate.Sub(inv.InvoiceDate)))
	applied := keel.NumericOf(cpa.AppliedAmountMinor.Sum()).NullIf(0)
	return orZero(keel.Scalar(postedApplications(cpa, cp).
		Join(inv, inv.ID.EqCol(cpa.InvoiceID)).
		Pick(keel.NumericOf(weighted.Sum()).Div(applied)).
		Where(append([]keel.Pred{cpa.AppliedAmountMinor.Gt(0)}, preds...)...)))
}

func overdueAt(inv *invoicedb.InvoicesTable, asOf int64) keel.Pred {
	return keel.And(inv.DueDate.IsNotNull(), inv.DueDate.Lt(asOf))
}

type balanceOverviewRecord struct {
	HasInvoiced          bool
	TotalOpenMinor       int64
	OverdueMinor         int64
	UnappliedCashMinor   int64
	DisputedOpenMinor    int64
	OpenInvoiceCount     int
	OverdueInvoiceCount  int
	DisputedInvoiceCount int
	AvgDaysPastDue       float64
	CurrentMinor         int64
	Days1To30Minor       int64
	Days31To60Minor      int64
	Days61To90Minor      int64
	DaysOver90Minor      int64
}

func (r *repository) GetBalanceOverview(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARBalanceOverview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARBalanceOverview, error) {
			inv := invoicedb.Invoices
			posted := invoicedb.Invoices.As("posted")
			cp := customerpaymentdb.Payments
			open := receivablesql.OpenMinor(inv)
			overdue := overdueAt(inv, req.AsOfDate)
			disputed := inv.DisputeStatus.Eq(invoice.DisputeStatusDisputed)
			buckets := invoiceAgingBuckets(inv, req.AsOfDate)
			rec, err := keel.IntoOne[balanceOverviewRecord](ctx, r.db.Keel(ctx), inv.
				Pick(
					keel.Exists(posted.Where(
						receivablesql.ReceivableBillType(posted),
						posted.PostedAt.IsNotNull(),
					)),
					orZero(open.Sum()),
					orZero(amountWhen(overdue, open).Sum()),
					orZero(keel.Scalar(cp.
						Pick(cp.UnappliedAmountMinor.Sum()).
						Where(cp.Status.Eq(customerpayment.StatusPosted)))),
					orZero(amountWhen(disputed, open).Sum()),
					keel.Count(),
					keel.Count().Filter(overdue),
					keel.Count().Filter(disputed),
					orZero(keel.CaseNumeric().
						When(overdue, inDays(keel.Int64(req.AsOfDate).Sub(inv.DueDate))).
						End().
						Avg()),
					orZero(buckets[0]),
					orZero(buckets[1]),
					orZero(buckets[2]),
					orZero(buckets[3]),
					orZero(buckets[4]),
				).
				Where(receivablesql.Open(inv)))
			if err != nil {
				return nil, fmt.Errorf("get ar balance overview: %w", err)
			}

			return &customerledgerports.ARBalanceOverview{
				HasInvoiced:          rec.HasInvoiced,
				TotalOpenMinor:       rec.TotalOpenMinor,
				OverdueMinor:         rec.OverdueMinor,
				UnappliedCashMinor:   rec.UnappliedCashMinor,
				DisputedOpenMinor:    rec.DisputedOpenMinor,
				OpenInvoiceCount:     rec.OpenInvoiceCount,
				OverdueInvoiceCount:  rec.OverdueInvoiceCount,
				DisputedInvoiceCount: rec.DisputedInvoiceCount,
				AvgDaysPastDue:       rec.AvgDaysPastDue,
				Buckets: customerledgerports.ARAgingBucketTotals{
					CurrentMinor:    rec.CurrentMinor,
					Days1To30Minor:  rec.Days1To30Minor,
					Days31To60Minor: rec.Days31To60Minor,
					Days61To90Minor: rec.Days61To90Minor,
					DaysOver90Minor: rec.DaysOver90Minor,
					TotalOpenMinor:  rec.TotalOpenMinor,
				},
			}, nil
		},
	)
}

type paymentStatsRecord struct {
	PostedTodayMinor      int64
	PostedTodayCount      int
	UnappliedCashMinor    int64
	UnappliedPaymentCount int
	ReversedLast30Minor   int64
	ReversedLast30Count   int
}

func (r *repository) GetPaymentStats(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARPaymentStats, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARPaymentStats, error) {
			dayStart := req.AsOfDate - (req.AsOfDate % secondsPerDay)
			cp := customerpaymentdb.Payments
			posted := cp.Status.Eq(customerpayment.StatusPosted)
			postedToday := keel.And(posted, cp.PaymentDate.Gte(dayStart))
			reversed := keel.And(
				cp.Status.Eq(customerpayment.StatusReversed),
				cp.ReversedAt.Gte(req.AsOfDate-30*secondsPerDay),
			)
			rec, err := keel.IntoOne[paymentStatsRecord](ctx, r.db.Keel(ctx), cp.Pick(
				orZero(cp.AmountMinor.Sum().Filter(postedToday)),
				keel.Count().Filter(postedToday),
				orZero(cp.UnappliedAmountMinor.Sum().Filter(posted)),
				keel.Count().Filter(keel.And(posted, cp.UnappliedAmountMinor.Gt(0))),
				orZero(cp.AmountMinor.Sum().Filter(reversed)),
				keel.Count().Filter(reversed),
			))
			if err != nil {
				return nil, fmt.Errorf("get ar payment stats: %w", err)
			}

			return &customerledgerports.ARPaymentStats{
				PostedTodayMinor:      rec.PostedTodayMinor,
				PostedTodayCount:      rec.PostedTodayCount,
				UnappliedCashMinor:    rec.UnappliedCashMinor,
				UnappliedPaymentCount: rec.UnappliedPaymentCount,
				ReversedLast30Minor:   rec.ReversedLast30Minor,
				ReversedLast30Count:   rec.ReversedLast30Count,
			}, nil
		},
	)
}

type draftInvoicesRecord struct {
	Count           int
	AmountMinor     int64
	OldestCreatedAt int64
}

func (r *repository) GetDraftInvoices(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDraftInvoices, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARDraftInvoices, error) {
			inv := invoicedb.Invoices
			rec, err := keel.IntoOne[draftInvoicesRecord](ctx, r.db.Keel(ctx), inv.
				Pick(keel.Count(), orZero(inv.TotalAmountMinor.Sum()), orZero(inv.CreatedAt.Min())).
				Where(
					inv.Status.Eq(invoice.StatusDraft),
					receivablesql.ReceivableBillType(inv),
					inv.CreatedAt.Lte(req.AsOfDate),
				))
			if err != nil {
				return nil, fmt.Errorf("get ar draft invoices: %w", err)
			}

			return &customerledgerports.ARDraftInvoices{
				Count:           rec.Count,
				AmountMinor:     rec.AmountMinor,
				OldestCreatedAt: rec.OldestCreatedAt,
			}, nil
		},
	)
}

type balancePointRecord struct {
	PeriodEnd      int64
	ARBalanceMinor int64
	BilledMinor    int64
}

func balanceSeriesQuery(req customerledgerports.ListARSeriesRequest) *keel.Query[*keel.Row] {
	at := seriesWindow(req).build()
	items := at.items
	points := at.points
	balances := keel.CTE("balances", []string{"n", "ar_balance_minor"}, items.Query().
		Pick(items.Col("n"), keel.Sum(items.Col("open_minor"))).
		Group(items.Col("n")))
	periodEnd := keel.Int64Of(points.Col(colPeriodEnd))
	inv := invoicedb.Invoices

	return points.Query().
		With(append(at.ctes, balances)...).
		LeftJoin(balances, balances.Col("n").EqCol(points.Col("n"))).
		Pick(
			periodEnd,
			keel.NumericOf(balances.Col("ar_balance_minor")).Coalesce(0),
			orZero(keel.Scalar(inv.
				Pick(inv.TotalAmountMinor.Sum()).
				Where(billedIn(inv, periodEnd.Sub(keel.Int64(trailingDSOWindow)), periodEnd)),
			)),
		).
		Order(periodEnd.Asc())
}

func (r *repository) ListBalanceSeries(
	ctx context.Context,
	req customerledgerports.ListARSeriesRequest,
) ([]*customerledgerports.ARBalancePoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARBalancePoint, error) {
			records, err := keel.Into[balancePointRecord](
				ctx,
				r.db.Keel(ctx),
				balanceSeriesQuery(req),
			)
			if err != nil {
				return nil, fmt.Errorf("list ar balance series: %w", err)
			}

			points := make([]*customerledgerports.ARBalancePoint, 0, len(records))
			for _, rec := range records {
				points = append(points, &customerledgerports.ARBalancePoint{
					PeriodEnd:      rec.PeriodEnd,
					ARBalanceMinor: rec.ARBalanceMinor,
					BilledMinor:    rec.BilledMinor,
				})
			}
			return points, nil
		},
	)
}

type agingTrendRecord struct {
	PeriodEnd       int64
	CurrentMinor    int64
	Days1To30Minor  int64
	Days31To60Minor int64
	Days61To90Minor int64
	DaysOver90Minor int64
	TotalOpenMinor  int64
}

func agingTrendQuery(req customerledgerports.ListARSeriesRequest) *keel.Query[*keel.Row] {
	at := seriesWindow(req).build()
	items := at.items
	points := at.points
	periodEnd := keel.Int64Of(points.Col(colPeriodEnd))
	open := keel.Int64Of(items.Col("open_minor"))
	buckets := agingBuckets(keel.Int64Of(items.Col(colDueDate)), periodEnd, open)

	return points.Query().
		With(at.ctes...).
		LeftJoin(items, items.Col("n").EqCol(points.Col("n"))).
		Pick(
			periodEnd,
			orZero(buckets[0]),
			orZero(buckets[1]),
			orZero(buckets[2]),
			orZero(buckets[3]),
			orZero(buckets[4]),
			orZero(open.Sum()),
		).
		Group(periodEnd).
		Order(periodEnd.Asc())
}

func (r *repository) ListAgingTrend(
	ctx context.Context,
	req customerledgerports.ListARSeriesRequest,
) ([]*customerledgerports.ARAgingTrendPoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARAgingTrendPoint, error) {
			records, err := keel.Into[agingTrendRecord](ctx, r.db.Keel(ctx), agingTrendQuery(req))
			if err != nil {
				return nil, fmt.Errorf("list ar aging trend: %w", err)
			}

			points := make([]*customerledgerports.ARAgingTrendPoint, 0, len(records))
			for _, rec := range records {
				points = append(points, &customerledgerports.ARAgingTrendPoint{
					PeriodEnd: rec.PeriodEnd,
					Buckets: customerledgerports.ARAgingBucketTotals{
						CurrentMinor:    rec.CurrentMinor,
						Days1To30Minor:  rec.Days1To30Minor,
						Days31To60Minor: rec.Days31To60Minor,
						Days61To90Minor: rec.Days61To90Minor,
						DaysOver90Minor: rec.DaysOver90Minor,
						TotalOpenMinor:  rec.TotalOpenMinor,
					},
				})
			}
			return points, nil
		},
	)
}

type cashFlowRecord struct {
	WeekStart            int64
	ExpectedMinor        int64
	OpenDueMinor         int64
	ActualMinor          int64
	FactorAdvancesMinor  int64
	ReserveReleasesMinor int64
	IsForecast           bool
}

func (r *repository) ListCashFlow(
	ctx context.Context,
	req customerledgerports.ListARCashFlowRequest,
) ([]*customerledgerports.ARCashFlowPoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARCashFlowPoint, error) {
			records, err := keel.Into[cashFlowRecord](ctx, r.db.Keel(ctx), cashFlowQuery(req))
			if err != nil {
				return nil, fmt.Errorf("list ar cash flow: %w", err)
			}

			points := make([]*customerledgerports.ARCashFlowPoint, 0, len(records))
			for _, rec := range records {
				points = append(points, &customerledgerports.ARCashFlowPoint{
					WeekStart:            rec.WeekStart,
					ExpectedMinor:        rec.ExpectedMinor,
					OpenDueMinor:         rec.OpenDueMinor,
					ActualMinor:          rec.ActualMinor,
					FactorAdvancesMinor:  rec.FactorAdvancesMinor,
					ReserveReleasesMinor: rec.ReserveReleasesMinor,
					IsForecast:           rec.IsForecast,
				})
			}
			return points, nil
		},
	)
}

func cashFlowQuery(req customerledgerports.ListARCashFlowRequest) *keel.Query[*keel.Row] {
	weeks := keel.CTE("weeks", []string{"week_start"}, keel.Rows(
		"week",
		[]string{"week_start"},
		keel.GenerateSeries(
			keel.Int64(int64(-req.PastWeeks)),
			keel.Int64(int64(req.FutureWeeks-1)),
		).Mul(keel.Int64(secondsPerWeek)).Add(keel.Int64(req.AsOfDate)),
	))
	weekStart := keel.Int64Of(weeks.Col("week_start"))
	weekEnd := weekStart.Add(keel.Int64(secondsPerWeek))
	inWeek := func(at keel.Expr) keel.Pred {
		return keel.And(
			keel.Int64Of(at).GteExpr(weekStart),
			keel.Int64Of(at).LtExpr(weekEnd),
		)
	}

	inv := invoicedb.Invoices
	cp := customerpaymentdb.Payments
	fsi := factoringsubmissiondb.SubmissionItems
	fs := factoringsubmissiondb.Submissions
	agr := factoringdb.Agreements
	reserved := factoringsubmissiondb.SubmissionItems.As("reserved")
	reservedInv := invoicedb.Invoices.As("reserved_inv")
	advanceAt := fs.SentAt.Add(agr.FundingDays.Mul(keel.Int64(secondsPerDay))).
		Greatest(keel.Int64(req.AsOfDate))
	reservedDue := keel.Int64Of(keel.Scalar(reservedInv.
		Pick(reservedInv.DueDate).
		Where(reservedInv.ID.EqCol(reserved.InvoiceID))))
	releaseAt := reservedDue.Greatest(keel.Int64(req.AsOfDate))

	return weeks.Query().With(weeks).
		Pick(
			weekStart,
			orZero(keel.Scalar(inv.
				Pick(inv.TotalAmountMinor.Sum()).
				Where(
					receivablesql.Receivable(inv),
					inWeek(inv.DueDate),
					keel.Not(factoringsql.LiveInvoicePred(inv)),
				))),
			orZero(keel.Scalar(inv.
				Pick(receivablesql.OpenMinor(inv).Sum()).
				Where(
					receivablesql.Open(inv),
					inWeek(inv.DueDate),
					keel.Not(factoringsql.LiveInvoicePred(inv)),
				))),
			orZero(keel.Scalar(cp.
				Pick(cp.AmountMinor.Sum()).
				Where(cp.Status.Eq(customerpayment.StatusPosted), inWeek(cp.PaymentDate)))),
			orZero(keel.Scalar(fsi.
				Pick(fsi.ExpectedAdvanceMinor.Sum()).
				Join(fs, fs.ID.EqCol(fsi.SubmissionID)).
				Join(agr, agr.ID.EqCol(fs.AgreementID)).
				Where(
					fsi.Status.Eq(factoringsubmission.SubmissionItemStatusPending),
					fs.Status.In(factoringsubmission.SubmissionStatusSent, factoringsubmission.SubmissionStatusPartially),
					fs.SentAt.IsNotNull(),
					inWeek(advanceAt),
				),
			)),
			orZero(keel.Scalar(reserved.
				Pick(reserved.ReserveMinor.Sum()).
				Where(
					reserved.Status.Eq(factoringsubmission.SubmissionItemStatusFunded),
					reservedDue.IsNotNull(),
					inWeek(releaseAt),
				))),
			weekStart.GteExpr(keel.Int64(req.AsOfDate)),
		).
		Order(weekStart.Asc())
}

type collectionTotalsRecord struct {
	BeginningOpenMinor       int64
	EndingOpenMinor          int64
	EndingCurrentMinor       int64
	CreditSalesMinor         int64
	CollectedMinor           int64
	AvgDaysToPay             float64
	ShortPayMinor            int64
	ShortPayApplicationCount int
	ApplicationCount         int
	DisputedInvoiceCount     int
	PostedInvoiceCount       int
}

func collectionTotalsQuery(periodStart, periodEnd int64) *keel.Query[*keel.Row] {
	at := instantWindow(periodStart).build()
	items := at.items
	bounds := keel.CTE("bounds", []string{"period_start", colPeriodEnd}, keel.Rows(
		"bound",
		[]string{"period_start", colPeriodEnd},
		keel.Int64(periodStart),
		keel.Int64(periodEnd),
	))
	start := keel.Int64Of(bounds.Col("period_start"))
	end := keel.Int64Of(bounds.Col(colPeriodEnd))
	inPeriod := func(ts keel.Expr) keel.Pred {
		return keel.And(keel.Int64Of(ts).GtExpr(start), keel.Int64Of(ts).LteExpr(end))
	}

	inv := invoicedb.Invoices.As("inv")
	cp := customerpaymentdb.Payments.As("cp")
	cpa := customerpaymentdb.Applications.As("cpa")
	applications := func(e keel.Expr, preds ...keel.Pred) keel.Expr {
		return keel.Scalar(postedApplications(cpa, cp).
			Pick(e).
			Where(append([]keel.Pred{inPeriod(cpa.CreatedAt)}, preds...)...))
	}
	open := receivablesql.OpenMinor(inv)

	return bounds.Query().
		With(append(at.ctes, bounds)...).
		Pick(
			orZero(keel.Scalar(items.Query().Pick(keel.Sum(items.Col("open_minor"))))),
			orZero(keel.Scalar(inv.Pick(open.Sum()).Where(receivablesql.Open(inv)))),
			orZero(keel.Scalar(inv.
				Pick(open.Sum()).
				Where(
					receivablesql.Open(inv),
					keel.Or(inv.DueDate.IsNull(), keel.Int64Of(inv.DueDate).GteExpr(end)),
				))),
			orZero(
				keel.Scalar(inv.Pick(inv.TotalAmountMinor.Sum()).Where(billedIn(inv, start, end))),
			),
			orZero(keel.Scalar(cp.
				Pick(cp.AmountMinor.Sum()).
				Where(cp.Status.Eq(customerpayment.StatusPosted), inPeriod(cp.PaymentDate)))),
			avgDaysToPay(cpa, cp, inv, inPeriod(cpa.CreatedAt)),
			orZero(applications(cpa.ShortPayAmountMinor.Sum())),
			applications(keel.Count(), cpa.ShortPayAmountMinor.Gt(0)),
			applications(keel.Count()),
			keel.Scalar(inv.
				Pick(keel.Count()).
				Where(
					receivablesql.Receivable(inv),
					inv.DisputeStatus.Eq(invoice.DisputeStatusDisputed),
					inPeriod(inv.InvoiceDate),
				)),
			keel.Scalar(inv.Pick(keel.Count()).Where(billedIn(inv, start, end))),
		)
}

func (r *repository) GetCollectionTotals(
	ctx context.Context,
	req customerledgerports.GetARCollectionMetricsRequest,
) (*customerledgerports.ARCollectionTotals, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARCollectionTotals, error) {
			periodStart := req.AsOfDate - int64(req.PeriodDays)*secondsPerDay
			rec, err := keel.IntoOne[collectionTotalsRecord](
				ctx,
				r.db.Keel(ctx),
				collectionTotalsQuery(periodStart, req.AsOfDate),
			)
			if err != nil {
				return nil, fmt.Errorf("get ar collection totals: %w", err)
			}

			return &customerledgerports.ARCollectionTotals{
				PeriodStart:              periodStart,
				PeriodEnd:                req.AsOfDate,
				BeginningOpenMinor:       rec.BeginningOpenMinor,
				EndingOpenMinor:          rec.EndingOpenMinor,
				EndingCurrentMinor:       rec.EndingCurrentMinor,
				CreditSalesMinor:         rec.CreditSalesMinor,
				CollectedMinor:           rec.CollectedMinor,
				AvgDaysToPay:             rec.AvgDaysToPay,
				ShortPayMinor:            rec.ShortPayMinor,
				ShortPayApplicationCount: rec.ShortPayApplicationCount,
				ApplicationCount:         rec.ApplicationCount,
				DisputedInvoiceCount:     rec.DisputedInvoiceCount,
				PostedInvoiceCount:       rec.PostedInvoiceCount,
			}, nil
		},
	)
}

type topOverdueRecord struct {
	CustomerID        string
	CustomerName      string
	OverdueMinor      int64
	TotalOpenMinor    int64
	OldestDaysPastDue int
	OpenInvoiceCount  int
}

func (r *repository) ListTopOverdueCustomers(
	ctx context.Context,
	req customerledgerports.ListARTopOverdueCustomersRequest,
) ([]*customerledgerports.ARTopOverdueCustomer, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARTopOverdueCustomer, error) {
			inv := invoicedb.Invoices
			open := receivablesql.OpenMinor(inv)
			overdue := overdueAt(inv, req.AsOfDate)
			overdueMinor := amountWhen(overdue, open).Sum()
			records, err := keel.Into[topOverdueRecord](ctx, r.db.Keel(ctx), inv.
				Pick(
					inv.CustomerID,
					inv.BillToName,
					overdueMinor,
					open.Sum(),
					orZero(keel.CaseInt64().
						When(overdue, pastDueDays(inv, req.AsOfDate)).
						End().
						Max()),
					keel.Count(),
				).
				Where(receivablesql.Open(inv)).
				Group(inv.CustomerID, inv.BillToName).
				Having(overdueMinor.Gt(0)).
				Order(overdueMinor.Desc()).
				Limit(req.Limit))
			if err != nil {
				return nil, fmt.Errorf("list ar top overdue customers: %w", err)
			}

			items := make([]*customerledgerports.ARTopOverdueCustomer, 0, len(records))
			for _, rec := range records {
				items = append(items, &customerledgerports.ARTopOverdueCustomer{
					CustomerID:        pulid.ID(rec.CustomerID),
					CustomerName:      rec.CustomerName,
					OverdueMinor:      rec.OverdueMinor,
					TotalOpenMinor:    rec.TotalOpenMinor,
					OldestDaysPastDue: rec.OldestDaysPastDue,
					OpenInvoiceCount:  rec.OpenInvoiceCount,
				})
			}
			return items, nil
		},
	)
}

type worklistRecord struct {
	InvoiceID             string
	CustomerID            string
	CustomerName          string
	InvoiceNumber         string
	DueDate               int64
	OpenAmountMinor       int64
	DaysPastDue           int
	IsDisputed            bool
	HasShortPay           bool
	OpenDisputeReasonCode string
	DisputedAmountMinor   int64
	DisputeOpenedAt       *int64
}

var worklistColumns = []string{
	colInvoiceID,
	"customer_id",
	"customer_name",
	"invoice_number",
	colDueDate,
	"open_amount_minor",
	"days_past_due",
	"is_disputed",
	"has_short_pay",
	"open_dispute_reason_code",
	"disputed_amount_minor",
	"dispute_opened_at",
}

func worklistQuery(
	req customerledgerports.ListARCollectionsWorklistRequest,
) *keel.Query[*keel.Row] {
	inv := invoicedb.Invoices
	od := invoicedisputedb.InvoiceDisputes.As("od")
	items := keel.CTE("items", worklistColumns, inv.
		LeftJoinLateral(od.
			Pick(od.ID, od.ReasonCode, od.DisputedAmountMinor, od.OpenedAt).
			Where(od.InvoiceID.EqCol(inv.ID), od.Status.Eq(invoicedispute.DisputeCaseStatusOpen)).
			Order(od.OpenedAt.Desc()).
			Limit(1)).
		Pick(
			inv.ID,
			inv.CustomerID,
			inv.BillToName,
			inv.Number,
			inv.DueDate.Coalesce(0),
			receivablesql.OpenMinor(inv),
			daysPastDue(inv, req.AsOfDate),
			keel.Or(inv.DisputeStatus.Eq(invoice.DisputeStatusDisputed), od.ID.IsNotNull()),
			hasShortPay(inv),
			od.ReasonCode.Coalesce(""),
			od.DisputedAmountMinor.Coalesce(0),
			od.OpenedAt,
		).
		Where(receivablesql.Open(inv)))

	picks := make([]keel.Expr, 0, len(worklistColumns))
	for _, name := range worklistColumns {
		picks = append(picks, items.Col(name))
	}
	daysPastDue := items.Col("days_past_due")
	return items.Query().With(items).
		Pick(picks...).
		Where(keel.Or(
			daysPastDue.Gte(15),
			keel.BoolOf(items.Col("is_disputed")).IsTrue(),
			keel.BoolOf(items.Col("has_short_pay")).IsTrue(),
		)).
		Order(daysPastDue.Desc(), items.Col("open_amount_minor").Desc()).
		Limit(req.Limit)
}

func (r *repository) ListCollectionsWorklist(
	ctx context.Context,
	req customerledgerports.ListARCollectionsWorklistRequest,
) ([]*customerledgerports.ARCollectionsWorklistItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARCollectionsWorklistItem, error) {
			records, err := keel.Into[worklistRecord](ctx, r.db.Keel(ctx), worklistQuery(req))
			if err != nil {
				return nil, fmt.Errorf("list ar collections worklist: %w", err)
			}

			items := make([]*customerledgerports.ARCollectionsWorklistItem, 0, len(records))
			for _, rec := range records {
				items = append(items, &customerledgerports.ARCollectionsWorklistItem{
					InvoiceID:       pulid.ID(rec.InvoiceID),
					CustomerID:      pulid.ID(rec.CustomerID),
					CustomerName:    rec.CustomerName,
					InvoiceNumber:   rec.InvoiceNumber,
					DueDate:         rec.DueDate,
					OpenAmountMinor: rec.OpenAmountMinor,
					DaysPastDue:     rec.DaysPastDue,
					IsDisputed:      rec.IsDisputed,
					HasShortPay:     rec.HasShortPay,

					OpenDisputeReasonCode: rec.OpenDisputeReasonCode,
					DisputedAmountMinor:   rec.DisputedAmountMinor,
					DisputeOpenedAt:       rec.DisputeOpenedAt,
				})
			}
			return items, nil
		},
	)
}

type customerSnapshotRecord struct {
	CustomerName          string
	CreditLimitMinor      int64
	HasCreditLimit        bool
	UnappliedCashMinor    int64
	OpenInvoiceCount      int
	OldestOpenInvoiceDate int64
	OldestDaysPastDue     int
	LastPaymentDate       int64
	LastPaymentMinor      int64
	AvgDaysToPay          float64
	BilledTrailing91Minor int64
}

type monthlyCollectionRecord struct {
	MonthStart  int64
	AmountMinor int64
}

func customerSnapshotQuery(
	req customerledgerports.GetARCustomerSnapshotRequest,
) *keel.Query[*customer.Customer] {
	cus := customerdb.Customers
	cbp := customerdb.CustomerBillingProfiles.As("cbp")
	cp := customerpaymentdb.Payments.As("cp")
	cpa := customerpaymentdb.Applications.As("cpa")
	inv := invoicedb.Invoices.As("inv")
	asOf := req.AsOfDate

	profile := func(e keel.Expr) keel.Expr {
		return keel.Scalar(cbp.Pick(e).Where(cbp.CustomerID.EqCol(cus.ID)))
	}
	payments := func() *keel.Query[*customerpayment.Payment] {
		return cp.Where(cp.CustomerID.EqCol(cus.ID), cp.Status.Eq(customerpayment.StatusPosted))
	}
	openInvoices := func(e keel.Expr) keel.Expr {
		return keel.Scalar(inv.Pick(e).Where(inv.CustomerID.EqCol(cus.ID), receivablesql.Open(inv)))
	}
	latestPayment := func(e keel.Expr) keel.Expr {
		return keel.Scalar(payments().Pick(e).Order(cp.PaymentDate.Desc()).Limit(1))
	}

	return cus.
		Pick(
			cus.Name,
			orZero(profile(keel.CastInt64(cbp.CreditLimit.Mul(keel.Numeric(100))))),
			keel.BoolOf(profile(cbp.CreditLimit.IsNotNull())).Coalesce(false),
			orZero(keel.Scalar(payments().Pick(cp.UnappliedAmountMinor.Sum()))),
			orZero(openInvoices(keel.Count())),
			orZero(openInvoices(inv.InvoiceDate.Min())),
			orZero(openInvoices(amountWhen(overdueAt(inv, asOf), pastDueDays(inv, asOf)).Max())),
			orZero(latestPayment(cp.PaymentDate)),
			orZero(latestPayment(cp.AmountMinor)),
			avgDaysToPay(
				cpa,
				cp,
				inv,
				cp.CustomerID.EqCol(cus.ID),
				cpa.CreatedAt.Gt(asOf-trailingYearWindow),
			),
			orZero(keel.Scalar(inv.
				Pick(inv.TotalAmountMinor.Sum()).
				Where(inv.CustomerID.EqCol(cus.ID), billedIn(inv, keel.Int64(asOf-trailingDSOWindow), keel.Int64(asOf))),
			)),
		).
		Where(cus.ID.Eq(req.CustomerID))
}

func customerMonthlyCollectionsQuery(
	req customerledgerports.GetARCustomerSnapshotRequest,
) *keel.Query[*customerpayment.Payment] {
	cp := customerpaymentdb.Payments
	month := keel.Epoch(keel.DateTrunc("month", keel.ToTimestamp(cp.PaymentDate)))
	return cp.
		Pick(month, cp.AmountMinor.Sum()).
		Where(
			cp.CustomerID.Eq(req.CustomerID),
			cp.Status.Eq(customerpayment.StatusPosted),
			cp.PaymentDate.Gt(req.AsOfDate-trailingYearWindow),
		).
		Group(month).
		Order(month.Asc())
}

func (r *repository) GetCustomerSnapshot(
	ctx context.Context,
	req customerledgerports.GetARCustomerSnapshotRequest,
) (*customerledgerports.ARCustomerSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARCustomerSnapshot, error) {
			aging, err := r.GetCustomerAging(
				ctx,
				customerledgerports.GetARCustomerAgingRequest(req),
			)
			if err != nil {
				return nil, err
			}

			rec, err := keel.IntoOne[customerSnapshotRecord](
				ctx,
				r.db.Keel(ctx),
				customerSnapshotQuery(req),
			)
			if err != nil {
				return nil, fmt.Errorf("get ar customer snapshot: %w", err)
			}

			monthly, err := keel.Into[monthlyCollectionRecord](
				ctx,
				r.db.Keel(ctx),
				customerMonthlyCollectionsQuery(req),
			)
			if err != nil {
				return nil, fmt.Errorf("get ar customer monthly collections: %w", err)
			}

			points := make([]*customerledgerports.ARMonthlyCollectionPoint, 0, len(monthly))
			for _, m := range monthly {
				points = append(points, &customerledgerports.ARMonthlyCollectionPoint{
					MonthStart:  m.MonthStart,
					AmountMinor: m.AmountMinor,
				})
			}

			snapshot := &customerledgerports.ARCustomerSnapshot{
				CustomerID:            req.CustomerID,
				CustomerName:          rec.CustomerName,
				UnappliedCashMinor:    rec.UnappliedCashMinor,
				CreditLimitMinor:      rec.CreditLimitMinor,
				HasCreditLimit:        rec.HasCreditLimit,
				OpenInvoiceCount:      rec.OpenInvoiceCount,
				OldestOpenInvoiceDate: rec.OldestOpenInvoiceDate,
				OldestDaysPastDue:     rec.OldestDaysPastDue,
				LastPaymentDate:       rec.LastPaymentDate,
				LastPaymentMinor:      rec.LastPaymentMinor,
				AvgDaysToPay:          rec.AvgDaysToPay,
				BilledTrailing91Minor: rec.BilledTrailing91Minor,
				MonthlyCollections:    points,
			}
			if aging != nil {
				snapshot.Buckets = aging.Buckets
				snapshot.TotalOpenMinor = aging.Buckets.TotalOpenMinor
				snapshot.OverdueMinor = aging.Buckets.TotalOpenMinor - aging.Buckets.CurrentMinor
				if snapshot.CustomerName == "" {
					snapshot.CustomerName = aging.CustomerName
				}
			}
			return snapshot, nil
		},
	)
}
