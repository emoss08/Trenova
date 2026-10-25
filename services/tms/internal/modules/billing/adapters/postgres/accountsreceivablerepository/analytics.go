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
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
)

const (
	secondsPerDay      = int64(86400)
	secondsPerWeek     = int64(604800)
	trailingDSOWindow  = 91 * secondsPerDay
	trailingYearWindow = 365 * secondsPerDay
)

const openInvoicePredicate = `
	  AND ` + receivablesql.OpenInvoice

func queryOne[T any](ctx context.Context, db *keel.DB, sql string, args ...any) (*T, error) {
	rows, err := keel.QueryAll(ctx, db, pgx.RowToAddrOfStructByName[T], sql, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, keel.ErrNotFound
	}
	return rows[0], nil
}

func inDays(seconds keel.DynExpr) keel.DynExpr {
	return seconds.Div(keel.DynLit(secondsPerDay).Cast(keel.TypeNumeric))
}

func orZero(e keel.Expr) keel.DynExpr {
	return keel.DynOf(e).Coalesce(0)
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
			buckets := agingBuckets(inv, req.AsOfDate)
			rec, err := keel.IntoOne[balanceOverviewRecord](ctx, r.db.Keel(ctx), inv.
				Pick(
					keel.Exists(posted.Where(
						receivablesql.ReceivableBillType(posted),
						posted.PostedAt.IsNotNull(),
					)),
					open.Sum().Coalesce(0),
					amountWhen(overdue, open).Sum().Coalesce(0),
					orZero(keel.Scalar(cp.
						Pick(cp.UnappliedAmountMinor.Sum()).
						Where(cp.Status.Eq(customerpayment.StatusPosted)))),
					amountWhen(disputed, open).Sum().Coalesce(0),
					keel.Count(),
					keel.Count().Filter(overdue),
					keel.Count().Filter(disputed),
					keel.DynCase().
						When(overdue, inDays(keel.DynLit(req.AsOfDate).Sub(inv.DueDate))).
						End().
						Avg().
						Coalesce(0),
					buckets[0].Coalesce(0),
					buckets[1].Coalesce(0),
					buckets[2].Coalesce(0),
					buckets[3].Coalesce(0),
					buckets[4].Coalesce(0),
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
	PeriodEnd      int64 `db:"period_end"`
	ARBalanceMinor int64 `db:"ar_balance_minor"`
	BilledMinor    int64 `db:"billed_minor"`
}

func (r *repository) ListBalanceSeries(
	ctx context.Context,
	req customerledgerports.ListARSeriesRequest,
) ([]*customerledgerports.ARBalancePoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARBalancePoint, error) {
			args := append(seriesWindow(req).args(), trailingDSOWindow)
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[balancePointRecord],
				`
		WITH`+openAsOfCTEs+`,
		balances AS (
			SELECT n, SUM(open_minor) AS ar_balance_minor
			FROM open_items
			GROUP BY n
		)
		SELECT
			p.period_end,
			COALESCE(b.ar_balance_minor, 0) AS ar_balance_minor,
			COALESCE((
				SELECT SUM(inv.total_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = $4
				  AND inv.business_unit_id = $5
				  AND inv.posted_at IS NOT NULL
				  AND (inv.voided_at IS NULL OR inv.voided_at > p.period_end)
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > p.period_end - $8::BIGINT
				  AND inv.invoice_date <= p.period_end
			), 0) AS billed_minor
		FROM points p
		LEFT JOIN balances b ON b.n = p.n
		ORDER BY p.period_end ASC`,
				args...,
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
	PeriodEnd       int64 `db:"period_end"`
	CurrentMinor    int64 `db:"current_minor"`
	Days1To30Minor  int64 `db:"days1_to30_minor"`
	Days31To60Minor int64 `db:"days31_to60_minor"`
	Days61To90Minor int64 `db:"days61_to90_minor"`
	DaysOver90Minor int64 `db:"days_over90_minor"`
	TotalOpenMinor  int64 `db:"total_open_minor"`
}

func (r *repository) ListAgingTrend(
	ctx context.Context,
	req customerledgerports.ListARSeriesRequest,
) ([]*customerledgerports.ARAgingTrendPoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARAgingTrendPoint, error) {
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[agingTrendRecord],
				`
		WITH`+openAsOfCTEs+`
		SELECT
			p.period_end,
			COALESCE(SUM(CASE WHEN oi.due_date IS NULL OR oi.due_date >= p.period_end THEN oi.open_minor ELSE 0 END), 0) AS current_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 BETWEEN 1 AND 30 THEN oi.open_minor ELSE 0 END), 0) AS days1_to30_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 BETWEEN 31 AND 60 THEN oi.open_minor ELSE 0 END), 0) AS days31_to60_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 BETWEEN 61 AND 90 THEN oi.open_minor ELSE 0 END), 0) AS days61_to90_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 > 90 THEN oi.open_minor ELSE 0 END), 0) AS days_over90_minor,
			COALESCE(SUM(oi.open_minor), 0) AS total_open_minor
		FROM points p
		LEFT JOIN open_items oi ON oi.n = p.n
		GROUP BY p.period_end
		ORDER BY p.period_end ASC`,
				seriesWindow(req).args()...,
			)
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

func cashFlowQuery(req customerledgerports.ListARCashFlowRequest) *keel.Query[keel.Row] {
	weeks := keel.CTE("weeks", []string{"week_start"}, keel.Rows(
		"week",
		[]string{"week_start"},
		keel.DynFunc(
			"generate_series",
			keel.DynLit(int64(-req.PastWeeks)).Cast(keel.TypeInt64),
			keel.DynLit(int64(req.FutureWeeks-1)).Cast(keel.TypeInt64),
		).Mul(secondsPerWeek).Add(req.AsOfDate),
	))
	weekStart := keel.DynOf(weeks.Col("week_start"))
	weekEnd := weekStart.Add(secondsPerWeek)
	inWeek := func(at keel.Expr) keel.Pred {
		return keel.And(keel.DynOf(at).Gte(weekStart), keel.DynOf(at).Lt(weekEnd))
	}

	inv := invoicedb.Invoices
	cp := customerpaymentdb.Payments
	fsi := factoringdb.SubmissionItems
	fs := factoringdb.Submissions
	agr := factoringdb.Agreements
	reserved := factoringdb.SubmissionItems.As("reserved")
	reservedInv := invoicedb.Invoices.As("reserved_inv")
	advanceAt := keel.DynFunc(
		"greatest",
		keel.DynOf(fs.SentAt).Add(keel.DynOf(agr.FundingDays).Mul(secondsPerDay)),
		keel.DynLit(req.AsOfDate).Cast(keel.TypeInt64),
	)
	releaseAt := keel.DynFunc(
		"greatest",
		reservedInv.DueDate,
		keel.DynLit(req.AsOfDate).Cast(keel.TypeInt64),
	)

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
					fsi.Status.Eq(factoring.SubmissionItemStatusPending),
					fs.Status.In(factoring.SubmissionStatusSent, factoring.SubmissionStatusPartially),
					fs.SentAt.IsNotNull(),
					inWeek(advanceAt),
				))),
			orZero(keel.Scalar(reserved.
				Pick(reserved.ReserveMinor.Sum()).
				Join(reservedInv, reservedInv.ID.EqCol(reserved.InvoiceID)).
				Where(
					reserved.Status.Eq(factoring.SubmissionItemStatusFunded),
					reservedInv.DueDate.IsNotNull(),
					inWeek(releaseAt),
				))),
			weekStart.Gte(keel.DynLit(req.AsOfDate).Cast(keel.TypeInt64)),
		).
		Order(weekStart.Asc())
}

type collectionTotalsRecord struct {
	BeginningOpenMinor       int64   `db:"beginning_open_minor"`
	EndingOpenMinor          int64   `db:"ending_open_minor"`
	EndingCurrentMinor       int64   `db:"ending_current_minor"`
	CreditSalesMinor         int64   `db:"credit_sales_minor"`
	CollectedMinor           int64   `db:"collected_minor"`
	AvgDaysToPay             float64 `db:"avg_days_to_pay"`
	ShortPayMinor            int64   `db:"short_pay_minor"`
	ShortPayApplicationCount int     `db:"short_pay_application_count"`
	ApplicationCount         int     `db:"application_count"`
	DisputedInvoiceCount     int     `db:"disputed_invoice_count"`
	PostedInvoiceCount       int     `db:"posted_invoice_count"`
}

const collectionTotalsSQL = `
		WITH bounds AS (SELECT $1::BIGINT AS period_start, $8::BIGINT AS period_end),` + openAsOfCTEs + `
		SELECT
			(SELECT COALESCE(SUM(oi.open_minor), 0) FROM open_items oi) AS beginning_open_minor,
			(SELECT COALESCE(SUM(inv.total_amount_minor - inv.applied_amount_minor), 0)
				FROM invoices inv
				WHERE inv.organization_id = $4
				  AND inv.business_unit_id = $5` + openInvoicePredicate + `
			) AS ending_open_minor,
			(SELECT COALESCE(SUM(inv.total_amount_minor - inv.applied_amount_minor), 0)
				FROM bounds, invoices inv
				WHERE inv.organization_id = $4
				  AND inv.business_unit_id = $5` + openInvoicePredicate + `
				  AND (inv.due_date IS NULL OR inv.due_date >= bounds.period_end)
			) AS ending_current_minor,
			(SELECT COALESCE(SUM(inv.total_amount_minor), 0)
				FROM bounds, invoices inv
				WHERE inv.organization_id = $4
				  AND inv.business_unit_id = $5
				  AND inv.posted_at IS NOT NULL
				  AND (inv.voided_at IS NULL OR inv.voided_at > bounds.period_end)
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > bounds.period_start
				  AND inv.invoice_date <= bounds.period_end
			) AS credit_sales_minor,
			(SELECT COALESCE(SUM(cp.amount_minor), 0)
				FROM bounds, customer_payments cp
				WHERE cp.organization_id = $4
				  AND cp.business_unit_id = $5
				  AND cp.status = 'Posted'
				  AND cp.payment_date > bounds.period_start
				  AND cp.payment_date <= bounds.period_end
			) AS collected_minor,
			(SELECT COALESCE(SUM(cpa.applied_amount_minor * ((cp.payment_date - inv.invoice_date) / 86400.0)) / NULLIF(SUM(cpa.applied_amount_minor), 0), 0)
				FROM bounds, customer_payment_applications cpa
				JOIN customer_payments cp
				  ON cp.id = cpa.customer_payment_id
				 AND cp.organization_id = cpa.organization_id
				 AND cp.business_unit_id = cpa.business_unit_id
				 AND cp.status = 'Posted'
				JOIN invoices inv
				  ON inv.id = cpa.invoice_id
				 AND inv.organization_id = cpa.organization_id
				 AND inv.business_unit_id = cpa.business_unit_id
				WHERE cpa.organization_id = $4
				  AND cpa.business_unit_id = $5
				  AND cpa.applied_amount_minor > 0
				  AND cpa.created_at > bounds.period_start
				  AND cpa.created_at <= bounds.period_end
			) AS avg_days_to_pay,
			(SELECT COALESCE(SUM(cpa.short_pay_amount_minor), 0)
				FROM bounds, customer_payment_applications cpa
				JOIN customer_payments cp
				  ON cp.id = cpa.customer_payment_id
				 AND cp.organization_id = cpa.organization_id
				 AND cp.business_unit_id = cpa.business_unit_id
				 AND cp.status = 'Posted'
				WHERE cpa.organization_id = $4
				  AND cpa.business_unit_id = $5
				  AND cpa.created_at > bounds.period_start
				  AND cpa.created_at <= bounds.period_end
			) AS short_pay_minor,
			(SELECT COUNT(*)
				FROM bounds, customer_payment_applications cpa
				JOIN customer_payments cp
				  ON cp.id = cpa.customer_payment_id
				 AND cp.organization_id = cpa.organization_id
				 AND cp.business_unit_id = cpa.business_unit_id
				 AND cp.status = 'Posted'
				WHERE cpa.organization_id = $4
				  AND cpa.business_unit_id = $5
				  AND cpa.short_pay_amount_minor > 0
				  AND cpa.created_at > bounds.period_start
				  AND cpa.created_at <= bounds.period_end
			) AS short_pay_application_count,
			(SELECT COUNT(*)
				FROM bounds, customer_payment_applications cpa
				JOIN customer_payments cp
				  ON cp.id = cpa.customer_payment_id
				 AND cp.organization_id = cpa.organization_id
				 AND cp.business_unit_id = cpa.business_unit_id
				 AND cp.status = 'Posted'
				WHERE cpa.organization_id = $4
				  AND cpa.business_unit_id = $5
				  AND cpa.created_at > bounds.period_start
				  AND cpa.created_at <= bounds.period_end
			) AS application_count,
			(SELECT COUNT(*)
				FROM bounds, invoices inv
				WHERE inv.organization_id = $4
				  AND inv.business_unit_id = $5
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.dispute_status = 'Disputed'
				  AND inv.invoice_date > bounds.period_start
				  AND inv.invoice_date <= bounds.period_end
			) AS disputed_invoice_count,
			(SELECT COUNT(*)
				FROM bounds, invoices inv
				WHERE inv.organization_id = $4
				  AND inv.business_unit_id = $5
				  AND inv.posted_at IS NOT NULL
				  AND (inv.voided_at IS NULL OR inv.voided_at > bounds.period_end)
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > bounds.period_start
				  AND inv.invoice_date <= bounds.period_end
			) AS posted_invoice_count`

func (r *repository) GetCollectionTotals(
	ctx context.Context,
	req customerledgerports.GetARCollectionMetricsRequest,
) (*customerledgerports.ARCollectionTotals, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARCollectionTotals, error) {
			periodStart := req.AsOfDate - int64(req.PeriodDays)*secondsPerDay
			rec, err := queryOne[collectionTotalsRecord](
				ctx,
				r.db.Keel(ctx),
				collectionTotalsSQL,
				append(instantWindow(req.TenantInfo, periodStart).args(), req.AsOfDate)...,
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
					keel.DynCase().
						When(overdue, pastDueDays(inv, req.AsOfDate)).
						End().
						Max().
						Coalesce(0),
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
	"invoice_id",
	"customer_id",
	"customer_name",
	"invoice_number",
	"due_date",
	"open_amount_minor",
	"days_past_due",
	"is_disputed",
	"has_short_pay",
	"open_dispute_reason_code",
	"disputed_amount_minor",
	"dispute_opened_at",
}

func worklistQuery(req customerledgerports.ListARCollectionsWorklistRequest) *keel.Query[keel.Row] {
	inv := invoicedb.Invoices
	od := invoicedb.InvoiceDisputes.As("od")
	items := keel.CTE("items", worklistColumns, inv.
		LeftJoinLateral(od.
			Pick(od.ID, od.ReasonCode, od.DisputedAmountMinor, od.OpenedAt).
			Where(od.InvoiceID.EqCol(inv.ID), od.Status.Eq(invoice.DisputeCaseStatusOpen)).
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
			keel.DynOf(items.Col("is_disputed")).IsTrue(),
			keel.DynOf(items.Col("has_short_pay")).IsTrue(),
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
) *keel.Query[customer.Customer] {
	cus := customerdb.Customers
	cbp := customerdb.CustomerBillingProfiles.As("cbp")
	cp := customerpaymentdb.Payments.As("cp")
	cpa := customerpaymentdb.Applications.As("cpa")
	inv := invoicedb.Invoices.As("inv")
	asOf := req.AsOfDate

	profile := func(e keel.Expr) keel.Expr {
		return keel.Scalar(cbp.Pick(e).Where(cbp.CustomerID.EqCol(cus.ID)))
	}
	payments := func() *keel.Query[customerpayment.Payment] {
		return cp.Where(cp.CustomerID.EqCol(cus.ID), cp.Status.Eq(customerpayment.StatusPosted))
	}
	openInvoices := func(e keel.Expr) keel.Expr {
		return keel.Scalar(inv.Pick(e).Where(inv.CustomerID.EqCol(cus.ID), receivablesql.Open(inv)))
	}
	latestPayment := func(e keel.Expr) keel.Expr {
		return keel.Scalar(payments().Pick(e).Order(cp.PaymentDate.Desc()).Limit(1))
	}
	weightedDays := keel.DynOf(cpa.AppliedAmountMinor).
		Mul(inDays(keel.DynOf(cp.PaymentDate).Sub(inv.InvoiceDate)))

	return cus.
		Pick(
			cus.Name,
			orZero(profile(keel.DynOf(cbp.CreditLimit).Mul(100).Cast(keel.TypeInt64))),
			keel.DynOf(profile(cbp.CreditLimit.IsNotNull())).Coalesce(false),
			orZero(keel.Scalar(payments().Pick(cp.UnappliedAmountMinor.Sum()))),
			orZero(openInvoices(keel.Count())),
			orZero(openInvoices(inv.InvoiceDate.Min())),
			orZero(openInvoices(amountWhen(overdueAt(inv, asOf), pastDueDays(inv, asOf)).Max())),
			orZero(latestPayment(cp.PaymentDate)),
			orZero(latestPayment(cp.AmountMinor)),
			orZero(keel.Scalar(cpa.
				Pick(weightedDays.Sum().Div(keel.DynOf(cpa.AppliedAmountMinor.Sum()).NullIf(0))).
				Join(cp, cp.ID.EqCol(cpa.CustomerPaymentID), cp.Status.Eq(customerpayment.StatusPosted)).
				Join(inv, inv.ID.EqCol(cpa.InvoiceID)).
				Where(
					cp.CustomerID.EqCol(cus.ID),
					cpa.AppliedAmountMinor.Gt(0),
					cpa.CreatedAt.Gt(asOf-trailingYearWindow),
				))),
			orZero(keel.Scalar(inv.
				Pick(inv.TotalAmountMinor.Sum()).
				Where(
					inv.CustomerID.EqCol(cus.ID),
					inv.PostedAt.IsNotNull(),
					keel.Or(inv.VoidedAt.IsNull(), inv.VoidedAt.Gt(asOf)),
					receivablesql.ReceivableBillType(inv),
					inv.InvoiceDate.Gt(asOf-trailingDSOWindow),
					inv.InvoiceDate.Lte(asOf),
				))),
		).
		Where(cus.ID.Eq(req.CustomerID))
}

func customerMonthlyCollectionsQuery(
	req customerledgerports.GetARCustomerSnapshotRequest,
) *keel.Query[customerpayment.Payment] {
	cp := customerpaymentdb.Payments
	month := keel.DynFunc(
		"date_part",
		keel.DynConst("epoch"),
		keel.DynFunc("date_trunc", keel.DynConst("month"), keel.DynFunc("to_timestamp", cp.PaymentDate)),
	).Cast(keel.TypeInt64)
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
