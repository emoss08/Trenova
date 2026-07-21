package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	secondsPerDay      = int64(86400)
	secondsPerWeek     = int64(604800)
	trailingDSOWindow  = 91 * secondsPerDay
	trailingYearWindow = 365 * secondsPerDay
)

const openInvoicePredicate = `
	  AND inv.status = 'Posted'
	  AND inv.bill_type IN ('Invoice', 'DebitMemo')
	  AND inv.total_amount_minor > inv.applied_amount_minor`

const appliedAsOfExpr = `(
		SELECT COALESCE(SUM(cpa.applied_amount_minor + cpa.short_pay_amount_minor), 0)
		FROM customer_payment_applications cpa
		JOIN customer_payments cp
		  ON cp.id = cpa.customer_payment_id
		 AND cp.organization_id = cpa.organization_id
		 AND cp.business_unit_id = cpa.business_unit_id
		WHERE cpa.invoice_id = inv.id
		  AND cpa.organization_id = inv.organization_id
		  AND cpa.business_unit_id = inv.business_unit_id
		  AND cpa.created_at <= p.period_end
		  AND (cp.reversed_at IS NULL OR cp.reversed_at > p.period_end)
	)`

type balanceOverviewRecord struct {
	TotalOpenMinor       int64   `bun:"total_open_minor"`
	OverdueMinor         int64   `bun:"overdue_minor"`
	UnappliedCashMinor   int64   `bun:"unapplied_cash_minor"`
	DisputedOpenMinor    int64   `bun:"disputed_open_minor"`
	OpenInvoiceCount     int     `bun:"open_invoice_count"`
	OverdueInvoiceCount  int     `bun:"overdue_invoice_count"`
	DisputedInvoiceCount int     `bun:"disputed_invoice_count"`
	AvgDaysPastDue       float64 `bun:"avg_days_past_due"`
	CurrentMinor         int64   `bun:"current_minor"`
	Days1To30Minor       int64   `bun:"days1_to30_minor"`
	Days31To60Minor      int64   `bun:"days31_to60_minor"`
	Days61To90Minor      int64   `bun:"days61_to90_minor"`
	DaysOver90Minor      int64   `bun:"days_over90_minor"`
}

func (r *repository) GetBalanceOverview(
	ctx context.Context,
	req repositories.GetARAnalyticsRequest,
) (*repositories.ARBalanceOverview, error) {
	rec := new(balanceOverviewRecord)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			COALESCE(SUM(inv.total_amount_minor - inv.applied_amount_minor), 0) AS total_open_minor,
			COALESCE(SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ? THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS overdue_minor,
			COALESCE((
				SELECT SUM(cp.unapplied_amount_minor)
				FROM customer_payments cp
				WHERE cp.organization_id = ?
				  AND cp.business_unit_id = ?
				  AND cp.status = 'Posted'
			), 0) AS unapplied_cash_minor,
			COALESCE(SUM(CASE WHEN inv.dispute_status = 'Disputed' THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS disputed_open_minor,
			COUNT(*) AS open_invoice_count,
			COUNT(*) FILTER (WHERE inv.due_date IS NOT NULL AND inv.due_date < ?) AS overdue_invoice_count,
			COUNT(*) FILTER (WHERE inv.dispute_status = 'Disputed') AS disputed_invoice_count,
			COALESCE(AVG(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ? THEN (? - inv.due_date) / 86400.0 END), 0) AS avg_days_past_due,
			COALESCE(SUM(CASE WHEN inv.due_date IS NULL OR inv.due_date >= ? THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS current_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 BETWEEN 1 AND 30 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days1_to30_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 BETWEEN 31 AND 60 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days31_to60_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 BETWEEN 61 AND 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days61_to90_minor,
			COALESCE(SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 > 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END), 0) AS days_over90_minor
		FROM invoices inv
		WHERE inv.organization_id = ?
		  AND inv.business_unit_id = ?`+openInvoicePredicate,
		req.AsOfDate,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.AsOfDate,
		req.AsOfDate, req.AsOfDate,
		req.AsOfDate,
		req.AsOfDate, req.AsOfDate,
		req.AsOfDate, req.AsOfDate,
		req.AsOfDate, req.AsOfDate,
		req.AsOfDate, req.AsOfDate,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
	).Scan(ctx, rec)
	if err != nil {
		return nil, fmt.Errorf("get ar balance overview: %w", err)
	}

	return &repositories.ARBalanceOverview{
		TotalOpenMinor:       rec.TotalOpenMinor,
		OverdueMinor:         rec.OverdueMinor,
		UnappliedCashMinor:   rec.UnappliedCashMinor,
		DisputedOpenMinor:    rec.DisputedOpenMinor,
		OpenInvoiceCount:     rec.OpenInvoiceCount,
		OverdueInvoiceCount:  rec.OverdueInvoiceCount,
		DisputedInvoiceCount: rec.DisputedInvoiceCount,
		AvgDaysPastDue:       rec.AvgDaysPastDue,
		Buckets: repositories.ARAgingBucketTotals{
			CurrentMinor:    rec.CurrentMinor,
			Days1To30Minor:  rec.Days1To30Minor,
			Days31To60Minor: rec.Days31To60Minor,
			Days61To90Minor: rec.Days61To90Minor,
			DaysOver90Minor: rec.DaysOver90Minor,
			TotalOpenMinor:  rec.TotalOpenMinor,
		},
	}, nil
}

type paymentStatsRecord struct {
	PostedTodayMinor      int64 `bun:"posted_today_minor"`
	PostedTodayCount      int   `bun:"posted_today_count"`
	UnappliedCashMinor    int64 `bun:"unapplied_cash_minor"`
	UnappliedPaymentCount int   `bun:"unapplied_payment_count"`
	ReversedLast30Minor   int64 `bun:"reversed_last30_minor"`
	ReversedLast30Count   int   `bun:"reversed_last30_count"`
}

func (r *repository) GetPaymentStats(
	ctx context.Context,
	req repositories.GetARAnalyticsRequest,
) (*repositories.ARPaymentStats, error) {
	dayStart := req.AsOfDate - (req.AsOfDate % secondsPerDay)
	rec := new(paymentStatsRecord)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			COALESCE(SUM(cp.amount_minor) FILTER (WHERE cp.status = 'Posted' AND cp.payment_date >= ?), 0) AS posted_today_minor,
			COUNT(*) FILTER (WHERE cp.status = 'Posted' AND cp.payment_date >= ?) AS posted_today_count,
			COALESCE(SUM(cp.unapplied_amount_minor) FILTER (WHERE cp.status = 'Posted'), 0) AS unapplied_cash_minor,
			COUNT(*) FILTER (WHERE cp.status = 'Posted' AND cp.unapplied_amount_minor > 0) AS unapplied_payment_count,
			COALESCE(SUM(cp.amount_minor) FILTER (WHERE cp.status = 'Reversed' AND cp.reversed_at >= ?), 0) AS reversed_last30_minor,
			COUNT(*) FILTER (WHERE cp.status = 'Reversed' AND cp.reversed_at >= ?) AS reversed_last30_count
		FROM customer_payments cp
		WHERE cp.organization_id = ?
		  AND cp.business_unit_id = ?`,
		dayStart, dayStart,
		req.AsOfDate-30*secondsPerDay, req.AsOfDate-30*secondsPerDay,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
	).Scan(ctx, rec)
	if err != nil {
		return nil, fmt.Errorf("get ar payment stats: %w", err)
	}

	return &repositories.ARPaymentStats{
		PostedTodayMinor:      rec.PostedTodayMinor,
		PostedTodayCount:      rec.PostedTodayCount,
		UnappliedCashMinor:    rec.UnappliedCashMinor,
		UnappliedPaymentCount: rec.UnappliedPaymentCount,
		ReversedLast30Minor:   rec.ReversedLast30Minor,
		ReversedLast30Count:   rec.ReversedLast30Count,
	}, nil
}

type balancePointRecord struct {
	PeriodEnd      int64 `bun:"period_end"`
	ARBalanceMinor int64 `bun:"ar_balance_minor"`
	BilledMinor    int64 `bun:"billed_minor"`
}

func (r *repository) ListBalanceSeries(
	ctx context.Context,
	req repositories.ListARSeriesRequest,
) ([]*repositories.ARBalancePoint, error) {
	records := make([]*balancePointRecord, 0, req.Weeks)
	err := r.db.DBForContext(ctx).NewRaw(`
		WITH points AS (
			SELECT (?::BIGINT - (n * ?::BIGINT))::BIGINT AS period_end
			FROM generate_series(0, ?::INT - 1) AS n
		)
		SELECT
			p.period_end,
			COALESCE((
				SELECT SUM(inv.total_amount_minor - applied.amount)
				FROM invoices inv
				CROSS JOIN LATERAL (SELECT `+appliedAsOfExpr+` AS amount) applied
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date <= p.period_end
				  AND inv.total_amount_minor > applied.amount
			), 0) AS ar_balance_minor,
			COALESCE((
				SELECT SUM(inv.total_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > p.period_end - ?::BIGINT
				  AND inv.invoice_date <= p.period_end
			), 0) AS billed_minor
		FROM points p
		ORDER BY p.period_end ASC`,
		req.AsOfDate, secondsPerWeek, req.Weeks,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		trailingDSOWindow,
	).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("list ar balance series: %w", err)
	}

	points := make([]*repositories.ARBalancePoint, 0, len(records))
	for _, rec := range records {
		points = append(points, &repositories.ARBalancePoint{
			PeriodEnd:      rec.PeriodEnd,
			ARBalanceMinor: rec.ARBalanceMinor,
			BilledMinor:    rec.BilledMinor,
		})
	}
	return points, nil
}

type agingTrendRecord struct {
	PeriodEnd       int64 `bun:"period_end"`
	CurrentMinor    int64 `bun:"current_minor"`
	Days1To30Minor  int64 `bun:"days1_to30_minor"`
	Days31To60Minor int64 `bun:"days31_to60_minor"`
	Days61To90Minor int64 `bun:"days61_to90_minor"`
	DaysOver90Minor int64 `bun:"days_over90_minor"`
	TotalOpenMinor  int64 `bun:"total_open_minor"`
}

func (r *repository) ListAgingTrend(
	ctx context.Context,
	req repositories.ListARSeriesRequest,
) ([]*repositories.ARAgingTrendPoint, error) {
	records := make([]*agingTrendRecord, 0, req.Weeks)
	err := r.db.DBForContext(ctx).NewRaw(`
		WITH points AS (
			SELECT (?::BIGINT - (n * ?::BIGINT))::BIGINT AS period_end
			FROM generate_series(0, ?::INT - 1) AS n
		),
		open_items AS (
			SELECT
				p.period_end,
				inv.due_date,
				inv.total_amount_minor - `+appliedAsOfExpr+` AS open_minor
			FROM points p
			JOIN invoices inv
			  ON inv.organization_id = ?
			 AND inv.business_unit_id = ?
			 AND inv.status = 'Posted'
			 AND inv.bill_type IN ('Invoice', 'DebitMemo')
			 AND inv.invoice_date <= p.period_end
		)
		SELECT
			p.period_end,
			COALESCE(SUM(CASE WHEN oi.due_date IS NULL OR oi.due_date >= p.period_end THEN oi.open_minor ELSE 0 END), 0) AS current_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 BETWEEN 1 AND 30 THEN oi.open_minor ELSE 0 END), 0) AS days1_to30_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 BETWEEN 31 AND 60 THEN oi.open_minor ELSE 0 END), 0) AS days31_to60_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 BETWEEN 61 AND 90 THEN oi.open_minor ELSE 0 END), 0) AS days61_to90_minor,
			COALESCE(SUM(CASE WHEN oi.due_date < p.period_end AND (p.period_end - oi.due_date) / 86400 > 90 THEN oi.open_minor ELSE 0 END), 0) AS days_over90_minor,
			COALESCE(SUM(oi.open_minor), 0) AS total_open_minor
		FROM points p
		LEFT JOIN open_items oi
		  ON oi.period_end = p.period_end
		 AND oi.open_minor > 0
		GROUP BY p.period_end
		ORDER BY p.period_end ASC`,
		req.AsOfDate, secondsPerWeek, req.Weeks,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
	).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("list ar aging trend: %w", err)
	}

	points := make([]*repositories.ARAgingTrendPoint, 0, len(records))
	for _, rec := range records {
		points = append(points, &repositories.ARAgingTrendPoint{
			PeriodEnd: rec.PeriodEnd,
			Buckets: repositories.ARAgingBucketTotals{
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
}

type cashFlowRecord struct {
	WeekStart     int64 `bun:"week_start"`
	ExpectedMinor int64 `bun:"expected_minor"`
	OpenDueMinor  int64 `bun:"open_due_minor"`
	ActualMinor   int64 `bun:"actual_minor"`
	IsForecast    bool  `bun:"is_forecast"`
}

func (r *repository) ListCashFlow(
	ctx context.Context,
	req repositories.ListARCashFlowRequest,
) ([]*repositories.ARCashFlowPoint, error) {
	records := make([]*cashFlowRecord, 0, req.PastWeeks+req.FutureWeeks)
	err := r.db.DBForContext(ctx).NewRaw(`
		WITH weeks AS (
			SELECT (?::BIGINT + (n * ?::BIGINT))::BIGINT AS week_start
			FROM generate_series(-?::INT, ?::INT - 1) AS n
		)
		SELECT
			w.week_start,
			COALESCE((
				SELECT SUM(inv.total_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.due_date >= w.week_start
				  AND inv.due_date < w.week_start + ?::BIGINT
			), 0) AS expected_minor,
			COALESCE((
				SELECT SUM(inv.total_amount_minor - inv.applied_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.total_amount_minor > inv.applied_amount_minor
				  AND inv.due_date >= w.week_start
				  AND inv.due_date < w.week_start + ?::BIGINT
			), 0) AS open_due_minor,
			COALESCE((
				SELECT SUM(cp.amount_minor)
				FROM customer_payments cp
				WHERE cp.organization_id = ?
				  AND cp.business_unit_id = ?
				  AND cp.status = 'Posted'
				  AND cp.payment_date >= w.week_start
				  AND cp.payment_date < w.week_start + ?::BIGINT
			), 0) AS actual_minor,
			(w.week_start >= ?::BIGINT) AS is_forecast
		FROM weeks w
		ORDER BY w.week_start ASC`,
		req.AsOfDate, secondsPerWeek, req.PastWeeks, req.FutureWeeks,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, secondsPerWeek,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, secondsPerWeek,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, secondsPerWeek,
		req.AsOfDate,
	).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("list ar cash flow: %w", err)
	}

	points := make([]*repositories.ARCashFlowPoint, 0, len(records))
	for _, rec := range records {
		points = append(points, &repositories.ARCashFlowPoint{
			WeekStart:     rec.WeekStart,
			ExpectedMinor: rec.ExpectedMinor,
			OpenDueMinor:  rec.OpenDueMinor,
			ActualMinor:   rec.ActualMinor,
			IsForecast:    rec.IsForecast,
		})
	}
	return points, nil
}

type collectionTotalsRecord struct {
	BeginningOpenMinor       int64   `bun:"beginning_open_minor"`
	EndingOpenMinor          int64   `bun:"ending_open_minor"`
	EndingCurrentMinor       int64   `bun:"ending_current_minor"`
	CreditSalesMinor         int64   `bun:"credit_sales_minor"`
	CollectedMinor           int64   `bun:"collected_minor"`
	AvgDaysToPay             float64 `bun:"avg_days_to_pay"`
	ShortPayMinor            int64   `bun:"short_pay_minor"`
	ShortPayApplicationCount int     `bun:"short_pay_application_count"`
	ApplicationCount         int     `bun:"application_count"`
	DisputedInvoiceCount     int     `bun:"disputed_invoice_count"`
	PostedInvoiceCount       int     `bun:"posted_invoice_count"`
}

const collectionTotalsSQL = `
		WITH bounds AS (SELECT ?::BIGINT AS period_start, ?::BIGINT AS period_end)
		SELECT
			(SELECT COALESCE(SUM(inv.total_amount_minor - applied.amount), 0)
				FROM bounds, LATERAL (SELECT bounds.period_start AS period_end) p,
				invoices inv
				CROSS JOIN LATERAL (SELECT ` + appliedAsOfExpr + ` AS amount) applied
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date <= p.period_end
				  AND inv.total_amount_minor > applied.amount
			) AS beginning_open_minor,
			(SELECT COALESCE(SUM(inv.total_amount_minor - inv.applied_amount_minor), 0)
				FROM invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?` + openInvoicePredicate + `
			) AS ending_open_minor,
			(SELECT COALESCE(SUM(inv.total_amount_minor - inv.applied_amount_minor), 0)
				FROM bounds, invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?` + openInvoicePredicate + `
				  AND (inv.due_date IS NULL OR inv.due_date >= bounds.period_end)
			) AS ending_current_minor,
			(SELECT COALESCE(SUM(inv.total_amount_minor), 0)
				FROM bounds, invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > bounds.period_start
				  AND inv.invoice_date <= bounds.period_end
			) AS credit_sales_minor,
			(SELECT COALESCE(SUM(cp.amount_minor), 0)
				FROM bounds, customer_payments cp
				WHERE cp.organization_id = ?
				  AND cp.business_unit_id = ?
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
				WHERE cpa.organization_id = ?
				  AND cpa.business_unit_id = ?
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
				WHERE cpa.organization_id = ?
				  AND cpa.business_unit_id = ?
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
				WHERE cpa.organization_id = ?
				  AND cpa.business_unit_id = ?
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
				WHERE cpa.organization_id = ?
				  AND cpa.business_unit_id = ?
				  AND cpa.created_at > bounds.period_start
				  AND cpa.created_at <= bounds.period_end
			) AS application_count,
			(SELECT COUNT(*)
				FROM bounds, invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.dispute_status = 'Disputed'
				  AND inv.invoice_date > bounds.period_start
				  AND inv.invoice_date <= bounds.period_end
			) AS disputed_invoice_count,
			(SELECT COUNT(*)
				FROM bounds, invoices inv
				WHERE inv.organization_id = ?
				  AND inv.business_unit_id = ?
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > bounds.period_start
				  AND inv.invoice_date <= bounds.period_end
			) AS posted_invoice_count`

func (r *repository) GetCollectionTotals(
	ctx context.Context,
	req repositories.GetARCollectionMetricsRequest,
) (*repositories.ARCollectionTotals, error) {
	periodStart := req.AsOfDate - int64(req.PeriodDays)*secondsPerDay
	rec := new(collectionTotalsRecord)
	err := r.db.DBForContext(ctx).NewRaw(collectionTotalsSQL,
		periodStart, req.AsOfDate,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
	).Scan(ctx, rec)
	if err != nil {
		return nil, fmt.Errorf("get ar collection totals: %w", err)
	}

	return &repositories.ARCollectionTotals{
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
}

type topOverdueRecord struct {
	CustomerID        string `bun:"customer_id"`
	CustomerName      string `bun:"customer_name"`
	OverdueMinor      int64  `bun:"overdue_minor"`
	TotalOpenMinor    int64  `bun:"total_open_minor"`
	OldestDaysPastDue int    `bun:"oldest_days_past_due"`
	OpenInvoiceCount  int    `bun:"open_invoice_count"`
}

func (r *repository) ListTopOverdueCustomers(
	ctx context.Context,
	req repositories.ListARTopOverdueCustomersRequest,
) ([]*repositories.ARTopOverdueCustomer, error) {
	records := make([]*topOverdueRecord, 0, req.Limit)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			inv.customer_id,
			inv.bill_to_name AS customer_name,
			SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ? THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS overdue_minor,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS total_open_minor,
			COALESCE(MAX(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ? THEN ((? - inv.due_date) / 86400)::INT END), 0) AS oldest_days_past_due,
			COUNT(*) AS open_invoice_count
		FROM invoices inv
		WHERE inv.organization_id = ?
		  AND inv.business_unit_id = ?`+openInvoicePredicate+`
		GROUP BY inv.customer_id, inv.bill_to_name
		HAVING SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ? THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) > 0
		ORDER BY overdue_minor DESC
		LIMIT ?`,
		req.AsOfDate, req.AsOfDate, req.AsOfDate,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.AsOfDate,
		req.Limit,
	).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("list ar top overdue customers: %w", err)
	}

	items := make([]*repositories.ARTopOverdueCustomer, 0, len(records))
	for _, rec := range records {
		items = append(items, &repositories.ARTopOverdueCustomer{
			CustomerID:        pulid.ID(rec.CustomerID),
			CustomerName:      rec.CustomerName,
			OverdueMinor:      rec.OverdueMinor,
			TotalOpenMinor:    rec.TotalOpenMinor,
			OldestDaysPastDue: rec.OldestDaysPastDue,
			OpenInvoiceCount:  rec.OpenInvoiceCount,
		})
	}
	return items, nil
}

type worklistRecord struct {
	InvoiceID       string `bun:"invoice_id"`
	CustomerID      string `bun:"customer_id"`
	CustomerName    string `bun:"customer_name"`
	InvoiceNumber   string `bun:"invoice_number"`
	DueDate         int64  `bun:"due_date"`
	OpenAmountMinor int64  `bun:"open_amount_minor"`
	DaysPastDue     int    `bun:"days_past_due"`
	IsDisputed      bool   `bun:"is_disputed"`
	HasShortPay     bool   `bun:"has_short_pay"`
}

func (r *repository) ListCollectionsWorklist(
	ctx context.Context,
	req repositories.ListARCollectionsWorklistRequest,
) ([]*repositories.ARCollectionsWorklistItem, error) {
	records := make([]*worklistRecord, 0, req.Limit)
	err := r.db.DBForContext(ctx).NewRaw(`
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
					WHEN inv.due_date IS NULL OR inv.due_date >= ? THEN 0
					ELSE GREATEST(((? - inv.due_date) / 86400)::INT, 0)
				END AS days_past_due,
				(inv.dispute_status = 'Disputed') AS is_disputed,
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
			WHERE inv.organization_id = ?
			  AND inv.business_unit_id = ?`+openInvoicePredicate+`
		) items
		WHERE items.days_past_due >= 15
		   OR items.is_disputed
		   OR items.has_short_pay
		ORDER BY items.days_past_due DESC, items.open_amount_minor DESC
		LIMIT ?`,
		req.AsOfDate, req.AsOfDate,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		req.Limit,
	).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("list ar collections worklist: %w", err)
	}

	items := make([]*repositories.ARCollectionsWorklistItem, 0, len(records))
	for _, rec := range records {
		items = append(items, &repositories.ARCollectionsWorklistItem{
			InvoiceID:       pulid.ID(rec.InvoiceID),
			CustomerID:      pulid.ID(rec.CustomerID),
			CustomerName:    rec.CustomerName,
			InvoiceNumber:   rec.InvoiceNumber,
			DueDate:         rec.DueDate,
			OpenAmountMinor: rec.OpenAmountMinor,
			DaysPastDue:     rec.DaysPastDue,
			IsDisputed:      rec.IsDisputed,
			HasShortPay:     rec.HasShortPay,
		})
	}
	return items, nil
}

type customerSnapshotRecord struct {
	CustomerName          string  `bun:"customer_name"`
	CreditLimitMinor      int64   `bun:"credit_limit_minor"`
	HasCreditLimit        bool    `bun:"has_credit_limit"`
	UnappliedCashMinor    int64   `bun:"unapplied_cash_minor"`
	OpenInvoiceCount      int     `bun:"open_invoice_count"`
	OldestOpenInvoiceDate int64   `bun:"oldest_open_invoice_date"`
	OldestDaysPastDue     int     `bun:"oldest_days_past_due"`
	LastPaymentDate       int64   `bun:"last_payment_date"`
	LastPaymentMinor      int64   `bun:"last_payment_minor"`
	AvgDaysToPay          float64 `bun:"avg_days_to_pay"`
	BilledTrailing91Minor int64   `bun:"billed_trailing91_minor"`
}

type monthlyCollectionRecord struct {
	MonthStart  int64 `bun:"month_start"`
	AmountMinor int64 `bun:"amount_minor"`
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
				SELECT MAX(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ? THEN ((? - inv.due_date) / 86400)::INT ELSE 0 END)
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
				  AND cpa.created_at > ? - ?::BIGINT
			), 0) AS avg_days_to_pay,
			COALESCE((
				SELECT SUM(inv.total_amount_minor)
				FROM invoices inv
				WHERE inv.organization_id = cus.organization_id
				  AND inv.business_unit_id = cus.business_unit_id
				  AND inv.customer_id = cus.id
				  AND inv.status = 'Posted'
				  AND inv.bill_type IN ('Invoice', 'DebitMemo')
				  AND inv.invoice_date > ? - ?::BIGINT
				  AND inv.invoice_date <= ?
			), 0) AS billed_trailing91_minor
		FROM customers cus
		WHERE cus.organization_id = ?
		  AND cus.business_unit_id = ?
		  AND cus.id = ?`

const customerMonthlyCollectionsSQL = `
		SELECT
			EXTRACT(EPOCH FROM date_trunc('month', to_timestamp(cp.payment_date)))::BIGINT AS month_start,
			SUM(cp.amount_minor) AS amount_minor
		FROM customer_payments cp
		WHERE cp.organization_id = ?
		  AND cp.business_unit_id = ?
		  AND cp.customer_id = ?
		  AND cp.status = 'Posted'
		  AND cp.payment_date > ? - ?::BIGINT
		GROUP BY 1
		ORDER BY 1 ASC`

func (r *repository) GetCustomerSnapshot(
	ctx context.Context,
	req repositories.GetARCustomerSnapshotRequest,
) (*repositories.ARCustomerSnapshot, error) {
	aging, err := r.GetCustomerAging(ctx, repositories.GetARCustomerAgingRequest(req))
	if err != nil {
		return nil, err
	}

	rec := new(customerSnapshotRecord)
	err = r.db.DBForContext(ctx).NewRaw(customerSnapshotSQL,
		req.AsOfDate, req.AsOfDate,
		req.AsOfDate, trailingYearWindow,
		req.AsOfDate, trailingDSOWindow, req.AsOfDate,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID,
	).Scan(ctx, rec)
	if err != nil {
		return nil, fmt.Errorf("get ar customer snapshot: %w", err)
	}

	monthly := make([]*monthlyCollectionRecord, 0, 12)
	err = r.db.DBForContext(ctx).NewRaw(customerMonthlyCollectionsSQL,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID,
		req.AsOfDate, trailingYearWindow,
	).Scan(ctx, &monthly)
	if err != nil {
		return nil, fmt.Errorf("get ar customer monthly collections: %w", err)
	}

	points := make([]*repositories.ARMonthlyCollectionPoint, 0, len(monthly))
	for _, m := range monthly {
		points = append(points, &repositories.ARMonthlyCollectionPoint{
			MonthStart:  m.MonthStart,
			AmountMinor: m.AmountMinor,
		})
	}

	snapshot := &repositories.ARCustomerSnapshot{
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
}
