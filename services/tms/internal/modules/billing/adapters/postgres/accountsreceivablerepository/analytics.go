package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringsql"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
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

type balanceOverviewRecord struct {
	HasInvoiced          bool    `db:"has_invoiced"`
	TotalOpenMinor       int64   `db:"total_open_minor"`
	OverdueMinor         int64   `db:"overdue_minor"`
	UnappliedCashMinor   int64   `db:"unapplied_cash_minor"`
	DisputedOpenMinor    int64   `db:"disputed_open_minor"`
	OpenInvoiceCount     int     `db:"open_invoice_count"`
	OverdueInvoiceCount  int     `db:"overdue_invoice_count"`
	DisputedInvoiceCount int     `db:"disputed_invoice_count"`
	AvgDaysPastDue       float64 `db:"avg_days_past_due"`
	CurrentMinor         int64   `db:"current_minor"`
	Days1To30Minor       int64   `db:"days1_to30_minor"`
	Days31To60Minor      int64   `db:"days31_to60_minor"`
	Days61To90Minor      int64   `db:"days61_to90_minor"`
	DaysOver90Minor      int64   `db:"days_over90_minor"`
}

func (r *repository) GetBalanceOverview(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARBalanceOverview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARBalanceOverview, error) {
			org, bu := tenantArgs(req.TenantInfo)
			rec, err := queryOne[balanceOverviewRecord](
				ctx,
				r.db.Keel(ctx),
				`
		SELECT
			EXISTS (
				SELECT 1
				FROM invoices posted
				WHERE posted.organization_id = $1
				  AND posted.business_unit_id = $2
				  AND posted.bill_type IN ('Invoice', 'DebitMemo')
				  AND posted.posted_at IS NOT NULL
			) AS has_invoiced,
			COALESCE(SUM(inv.total_amount_minor - inv.applied_amount_minor), 0) AS total_open_minor,
			COALESCE(SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $3 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS overdue_minor,
			COALESCE((
				SELECT SUM(cp.unapplied_amount_minor)
				FROM customer_payments cp
				WHERE cp.organization_id = $1
				  AND cp.business_unit_id = $2
				  AND cp.status = 'Posted'
			), 0) AS unapplied_cash_minor,
			COALESCE(SUM(CASE WHEN inv.dispute_status = 'Disputed' THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS disputed_open_minor,
			COUNT(*) AS open_invoice_count,
			COUNT(*) FILTER (WHERE inv.due_date IS NOT NULL AND inv.due_date < $3) AS overdue_invoice_count,
			COUNT(*) FILTER (WHERE inv.dispute_status = 'Disputed') AS disputed_invoice_count,
			COALESCE(AVG(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $3 THEN ($3 - inv.due_date) / 86400.0 END), 0) AS avg_days_past_due,
			COALESCE(SUM(CASE WHEN inv.due_date IS NULL OR inv.due_date >= $3 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS current_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < $3 AND ($3 - inv.due_date) / 86400 BETWEEN 1 AND 30 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days1_to30_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < $3 AND ($3 - inv.due_date) / 86400 BETWEEN 31 AND 60 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days31_to60_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < $3 AND ($3 - inv.due_date) / 86400 BETWEEN 61 AND 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days61_to90_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < $3 AND ($3 - inv.due_date) / 86400 > 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days_over90_minor
		FROM invoices inv
		WHERE inv.organization_id = $1
		  AND inv.business_unit_id = $2`+openInvoicePredicate,
				org, bu, req.AsOfDate,
			)
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
	PostedTodayMinor      int64 `db:"posted_today_minor"`
	PostedTodayCount      int   `db:"posted_today_count"`
	UnappliedCashMinor    int64 `db:"unapplied_cash_minor"`
	UnappliedPaymentCount int   `db:"unapplied_payment_count"`
	ReversedLast30Minor   int64 `db:"reversed_last30_minor"`
	ReversedLast30Count   int   `db:"reversed_last30_count"`
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
			org, bu := tenantArgs(req.TenantInfo)
			rec, err := queryOne[paymentStatsRecord](
				ctx,
				r.db.Keel(ctx),
				`
		SELECT
			COALESCE(SUM(cp.amount_minor) FILTER (WHERE cp.status = 'Posted' AND cp.payment_date >= $1), 0) AS posted_today_minor,
			COUNT(*) FILTER (WHERE cp.status = 'Posted' AND cp.payment_date >= $1) AS posted_today_count,
			COALESCE(SUM(cp.unapplied_amount_minor) FILTER (WHERE cp.status = 'Posted'), 0) AS unapplied_cash_minor,
			COUNT(*) FILTER (WHERE cp.status = 'Posted' AND cp.unapplied_amount_minor > 0) AS unapplied_payment_count,
			COALESCE(SUM(cp.amount_minor) FILTER (WHERE cp.status = 'Reversed' AND cp.reversed_at >= $2), 0) AS reversed_last30_minor,
			COUNT(*) FILTER (WHERE cp.status = 'Reversed' AND cp.reversed_at >= $2) AS reversed_last30_count
		FROM customer_payments cp
		WHERE cp.organization_id = $3
		  AND cp.business_unit_id = $4`,
				dayStart, req.AsOfDate-30*secondsPerDay, org, bu,
			)
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
	Count           int   `db:"draft_count"`
	AmountMinor     int64 `db:"draft_amount_minor"`
	OldestCreatedAt int64 `db:"oldest_created_at"`
}

func (r *repository) GetDraftInvoices(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDraftInvoices, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARDraftInvoices, error) {
			org, bu := tenantArgs(req.TenantInfo)
			rec, err := queryOne[draftInvoicesRecord](
				ctx,
				r.db.Keel(ctx),
				`
		SELECT
			COUNT(*) AS draft_count,
			COALESCE(SUM(inv.total_amount_minor), 0) AS draft_amount_minor,
			COALESCE(MIN(inv.created_at), 0) AS oldest_created_at
		FROM invoices inv
		WHERE inv.organization_id = $1
		  AND inv.business_unit_id = $2
		  AND inv.status = 'Draft'
		  AND inv.bill_type IN ('Invoice', 'DebitMemo')
		  AND inv.created_at <= $3`,
				org, bu, req.AsOfDate,
			)
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
	WeekStart            int64 `db:"week_start"`
	ExpectedMinor        int64 `db:"expected_minor"`
	OpenDueMinor         int64 `db:"open_due_minor"`
	ActualMinor          int64 `db:"actual_minor"`
	FactorAdvancesMinor  int64 `db:"factor_advances_minor"`
	ReserveReleasesMinor int64 `db:"reserve_releases_minor"`
	IsForecast           bool  `db:"is_forecast"`
}

func (r *repository) ListCashFlow(
	ctx context.Context,
	req customerledgerports.ListARCashFlowRequest,
) ([]*customerledgerports.ARCashFlowPoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARCashFlowPoint, error) {
			org, bu := tenantArgs(req.TenantInfo)
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[cashFlowRecord],
				cashFlowSQL,
				req.AsOfDate, secondsPerWeek, req.PastWeeks, req.FutureWeeks, org, bu,
			)
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

var cashFlowSQL = `
		WITH weeks AS (
			SELECT ($1::BIGINT + (n * $2::BIGINT))::BIGINT AS week_start
			FROM generate_series(-$3::INT, $4::INT - 1) AS n
		)
		SELECT
			w.week_start,
			COALESCE((
				SELECT SUM(inv.total_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = $5
				  AND inv.business_unit_id = $6
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.due_date >= w.week_start
				  AND inv.due_date < w.week_start + $2::BIGINT
				  AND NOT ` + factoringsql.LiveInvoice() + `
			), 0) AS expected_minor,
			COALESCE((
				SELECT SUM(inv.total_amount_minor - inv.applied_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = $5
				  AND inv.business_unit_id = $6
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.total_amount_minor > inv.applied_amount_minor
				  AND inv.due_date >= w.week_start
				  AND inv.due_date < w.week_start + $2::BIGINT
				  AND NOT ` + factoringsql.LiveInvoice() + `
			), 0) AS open_due_minor,
			COALESCE((
				SELECT SUM(cp.amount_minor)
				FROM customer_payments cp
				WHERE cp.organization_id = $5
				  AND cp.business_unit_id = $6
				  AND cp.status = 'Posted'
				  AND cp.payment_date >= w.week_start
				  AND cp.payment_date < w.week_start + $2::BIGINT
			), 0) AS actual_minor,
			COALESCE((
				SELECT SUM(fsi.expected_advance_minor)
				FROM factoring_submission_items fsi
				JOIN factoring_submissions fs
				  ON fs.id = fsi.submission_id
				  AND fs.organization_id = fsi.organization_id
				  AND fs.business_unit_id = fsi.business_unit_id
				JOIN factoring_agreements agr
				  ON agr.id = fs.agreement_id
				  AND agr.organization_id = fs.organization_id
				  AND agr.business_unit_id = fs.business_unit_id
				WHERE fsi.organization_id = $5
				  AND fsi.business_unit_id = $6
				  AND fsi.status = 'Pending'
				  AND fs.status IN ('Sent', 'PartiallyFunded')
				  AND fs.sent_at IS NOT NULL
				  AND GREATEST(fs.sent_at + agr.funding_days * 86400, $1::BIGINT) >= w.week_start
				  AND GREATEST(fs.sent_at + agr.funding_days * 86400, $1::BIGINT)
				    < w.week_start + $2::BIGINT
			), 0) AS factor_advances_minor,
			COALESCE((
				SELECT SUM(fsi.reserve_minor)
				FROM factoring_submission_items fsi
				JOIN invoices inv
				  ON inv.id = fsi.invoice_id
				  AND inv.organization_id = fsi.organization_id
				  AND inv.business_unit_id = fsi.business_unit_id
				WHERE fsi.organization_id = $5
				  AND fsi.business_unit_id = $6
				  AND fsi.status = 'Funded'
				  AND inv.due_date IS NOT NULL
				  AND GREATEST(inv.due_date, $1::BIGINT) >= w.week_start
				  AND GREATEST(inv.due_date, $1::BIGINT) < w.week_start + $2::BIGINT
			), 0) AS reserve_releases_minor,
			(w.week_start >= $1::BIGINT) AS is_forecast
		FROM weeks w
		ORDER BY w.week_start ASC`

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
	CustomerID        string `db:"customer_id"`
	CustomerName      string `db:"customer_name"`
	OverdueMinor      int64  `db:"overdue_minor"`
	TotalOpenMinor    int64  `db:"total_open_minor"`
	OldestDaysPastDue int    `db:"oldest_days_past_due"`
	OpenInvoiceCount  int    `db:"open_invoice_count"`
}

func (r *repository) ListTopOverdueCustomers(
	ctx context.Context,
	req customerledgerports.ListARTopOverdueCustomersRequest,
) ([]*customerledgerports.ARTopOverdueCustomer, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARTopOverdueCustomer, error) {
			org, bu := tenantArgs(req.TenantInfo)
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[topOverdueRecord],
				`
		SELECT
			inv.customer_id,
			inv.bill_to_name AS customer_name,
			SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $1 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS overdue_minor,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS total_open_minor,
			COALESCE(MAX(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $1 THEN (($1 - inv.due_date) / 86400)::INT END), 0) AS oldest_days_past_due,
			COUNT(*) AS open_invoice_count
		FROM invoices inv
		WHERE inv.organization_id = $2
		  AND inv.business_unit_id = $3`+openInvoicePredicate+`
		GROUP BY inv.customer_id, inv.bill_to_name
		HAVING SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $1 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) > 0
		ORDER BY overdue_minor DESC
		LIMIT $4`,
				req.AsOfDate, org, bu, req.Limit,
			)
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
	InvoiceID             string `db:"invoice_id"`
	CustomerID            string `db:"customer_id"`
	CustomerName          string `db:"customer_name"`
	InvoiceNumber         string `db:"invoice_number"`
	DueDate               int64  `db:"due_date"`
	OpenAmountMinor       int64  `db:"open_amount_minor"`
	DaysPastDue           int    `db:"days_past_due"`
	IsDisputed            bool   `db:"is_disputed"`
	HasShortPay           bool   `db:"has_short_pay"`
	OpenDisputeReasonCode string `db:"open_dispute_reason_code"`
	DisputedAmountMinor   int64  `db:"disputed_amount_minor"`
	DisputeOpenedAt       *int64 `db:"dispute_opened_at"`
}

func (r *repository) ListCollectionsWorklist(
	ctx context.Context,
	req customerledgerports.ListARCollectionsWorklistRequest,
) ([]*customerledgerports.ARCollectionsWorklistItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARCollectionsWorklistItem, error) {
			org, bu := tenantArgs(req.TenantInfo)
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[worklistRecord],
				`
		SELECT *
		FROM (
			SELECT
				inv.id AS invoice_id,
				inv.customer_id,
				inv.bill_to_name AS customer_name,
				inv.number AS invoice_number,
				COALESCE(inv.due_date, 0) AS due_date,
				(inv.total_amount_minor - inv.applied_amount_minor) AS open_amount_minor,
				CASE
					WHEN inv.due_date IS NULL OR inv.due_date >= $1 THEN 0
					ELSE GREATEST((($1 - inv.due_date) / 86400)::INT, 0)
				END AS days_past_due,
				(inv.dispute_status = 'Disputed' OR od.id IS NOT NULL) AS is_disputed,
				COALESCE(od.reason_code, '') AS open_dispute_reason_code,
				COALESCE(od.disputed_amount_minor, 0) AS disputed_amount_minor,
				od.opened_at AS dispute_opened_at,
				EXISTS (
					SELECT 1
					FROM customer_payment_applications cpa
					JOIN customer_payments cp
					  ON cp.id = cpa.customer_payment_id
					 AND cp.organization_id = cpa.organization_id
					 AND cp.business_unit_id = cpa.business_unit_id
					 AND cp.status = 'Posted'
					WHERE cpa.invoice_id = inv.id
					  AND cpa.organization_id = inv.organization_id
					  AND cpa.business_unit_id = inv.business_unit_id
					  AND cpa.short_pay_amount_minor > 0
				) AS has_short_pay
			FROM invoices inv
			LEFT JOIN LATERAL (
				SELECT idsp.id, idsp.reason_code, idsp.disputed_amount_minor, idsp.opened_at
				FROM invoice_disputes idsp
				WHERE idsp.invoice_id = inv.id
				  AND idsp.organization_id = inv.organization_id
				  AND idsp.business_unit_id = inv.business_unit_id
				  AND idsp.status = 'Open'
				ORDER BY idsp.opened_at DESC
				LIMIT 1
			) od ON TRUE
			WHERE inv.organization_id = $2
			  AND inv.business_unit_id = $3`+openInvoicePredicate+`
		) items
		WHERE items.days_past_due >= 15
		   OR items.is_disputed
		   OR items.has_short_pay
		ORDER BY items.days_past_due DESC, items.open_amount_minor DESC
		LIMIT $4`,
				req.AsOfDate, org, bu, req.Limit,
			)
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
	CustomerName          string  `db:"customer_name"`
	CreditLimitMinor      int64   `db:"credit_limit_minor"`
	HasCreditLimit        bool    `db:"has_credit_limit"`
	UnappliedCashMinor    int64   `db:"unapplied_cash_minor"`
	OpenInvoiceCount      int     `db:"open_invoice_count"`
	OldestOpenInvoiceDate int64   `db:"oldest_open_invoice_date"`
	OldestDaysPastDue     int     `db:"oldest_days_past_due"`
	LastPaymentDate       int64   `db:"last_payment_date"`
	LastPaymentMinor      int64   `db:"last_payment_minor"`
	AvgDaysToPay          float64 `db:"avg_days_to_pay"`
	BilledTrailing91Minor int64   `db:"billed_trailing91_minor"`
}

type monthlyCollectionRecord struct {
	MonthStart  int64 `db:"month_start"`
	AmountMinor int64 `db:"amount_minor"`
}

const customerSnapshotSQL = `
		SELECT
			cus.name AS customer_name,
			COALESCE((
				SELECT (cbp.credit_limit * 100)::BIGINT
				FROM customer_billing_profiles cbp
				WHERE cbp.customer_id = cus.id
				  AND cbp.organization_id = cus.organization_id
				  AND cbp.business_unit_id = cus.business_unit_id
			), 0) AS credit_limit_minor,
			COALESCE((
				SELECT cbp.credit_limit IS NOT NULL
				FROM customer_billing_profiles cbp
				WHERE cbp.customer_id = cus.id
				  AND cbp.organization_id = cus.organization_id
				  AND cbp.business_unit_id = cus.business_unit_id
			), FALSE) AS has_credit_limit,
			COALESCE((
				SELECT SUM(cp.unapplied_amount_minor)
				FROM customer_payments cp
				WHERE cp.organization_id = cus.organization_id
				  AND cp.business_unit_id = cus.business_unit_id
				  AND cp.customer_id = cus.id
				  AND cp.status = 'Posted'
			), 0) AS unapplied_cash_minor,
			COALESCE((
				SELECT COUNT(*)
				FROM invoices inv
				WHERE inv.organization_id = cus.organization_id
				  AND inv.business_unit_id = cus.business_unit_id
				  AND inv.customer_id = cus.id` + openInvoicePredicate + `
			), 0) AS open_invoice_count,
			COALESCE((
				SELECT MIN(inv.invoice_date)
				FROM invoices inv
				WHERE inv.organization_id = cus.organization_id
				  AND inv.business_unit_id = cus.business_unit_id
				  AND inv.customer_id = cus.id` + openInvoicePredicate + `
			), 0) AS oldest_open_invoice_date,
			COALESCE((
				SELECT MAX(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $1 THEN (($1 - inv.due_date) / 86400)::INT ELSE 0 END)
				FROM invoices inv
				WHERE inv.organization_id = cus.organization_id
				  AND inv.business_unit_id = cus.business_unit_id
				  AND inv.customer_id = cus.id` + openInvoicePredicate + `
			), 0) AS oldest_days_past_due,
			COALESCE((
				SELECT cp.payment_date
				FROM customer_payments cp
				WHERE cp.organization_id = cus.organization_id
				  AND cp.business_unit_id = cus.business_unit_id
				  AND cp.customer_id = cus.id
				  AND cp.status = 'Posted'
				ORDER BY cp.payment_date DESC
				LIMIT 1
			), 0) AS last_payment_date,
			COALESCE((
				SELECT cp.amount_minor
				FROM customer_payments cp
				WHERE cp.organization_id = cus.organization_id
				  AND cp.business_unit_id = cus.business_unit_id
				  AND cp.customer_id = cus.id
				  AND cp.status = 'Posted'
				ORDER BY cp.payment_date DESC
				LIMIT 1
			), 0) AS last_payment_minor,
			COALESCE((
				SELECT SUM(cpa.applied_amount_minor * ((cp.payment_date - inv.invoice_date) / 86400.0)) / NULLIF(SUM(cpa.applied_amount_minor), 0)
				FROM customer_payment_applications cpa
				JOIN customer_payments cp
				  ON cp.id = cpa.customer_payment_id
				 AND cp.organization_id = cpa.organization_id
				 AND cp.business_unit_id = cpa.business_unit_id
				 AND cp.status = 'Posted'
				JOIN invoices inv
				  ON inv.id = cpa.invoice_id
				 AND inv.organization_id = cpa.organization_id
				 AND inv.business_unit_id = cpa.business_unit_id
				WHERE cpa.organization_id = cus.organization_id
				  AND cpa.business_unit_id = cus.business_unit_id
				  AND cp.customer_id = cus.id
				  AND cpa.applied_amount_minor > 0
				  AND cpa.created_at > $1 - $2::BIGINT
			), 0) AS avg_days_to_pay,
			COALESCE((
				SELECT SUM(inv.total_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = cus.organization_id
				  AND inv.business_unit_id = cus.business_unit_id
				  AND inv.customer_id = cus.id
				  AND inv.posted_at IS NOT NULL
				  AND (inv.voided_at IS NULL OR inv.voided_at > $1)
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > $1 - $3::BIGINT
				  AND inv.invoice_date <= $1
			), 0) AS billed_trailing91_minor
		FROM customers cus
		WHERE cus.organization_id = $4
		  AND cus.business_unit_id = $5
		  AND cus.id = $6`

const customerMonthlyCollectionsSQL = `
		SELECT
			EXTRACT(EPOCH FROM date_trunc('month', to_timestamp(cp.payment_date)))::BIGINT AS month_start,
			SUM(cp.amount_minor) AS amount_minor
		FROM customer_payments cp
		WHERE cp.organization_id = $1
		  AND cp.business_unit_id = $2
		  AND cp.customer_id = $3
		  AND cp.status = 'Posted'
		  AND cp.payment_date > $4 - $5::BIGINT
		GROUP BY 1
		ORDER BY 1 ASC`

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

			org, bu := tenantArgs(req.TenantInfo)
			rec, err := queryOne[customerSnapshotRecord](
				ctx,
				r.db.Keel(ctx),
				customerSnapshotSQL,
				req.AsOfDate, trailingYearWindow, trailingDSOWindow, org, bu, req.CustomerID.String(),
			)
			if err != nil {
				return nil, fmt.Errorf("get ar customer snapshot: %w", err)
			}

			monthly, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[monthlyCollectionRecord],
				customerMonthlyCollectionsSQL,
				org, bu, req.CustomerID.String(), req.AsOfDate, trailingYearWindow,
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
