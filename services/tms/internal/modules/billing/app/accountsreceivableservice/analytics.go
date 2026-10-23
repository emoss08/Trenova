package accountsreceivableservice

import (
	"context"
	"math"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	defaultTrendWeeks        = 13
	maxTrendWeeks            = 52
	dsoWindowDays            = 91.0
	collectionPeriodDays     = customerledger.CollectionWindowDays
	kpiDeltaWeeks            = 5
	defaultTopOverdueLimit   = 10
	defaultWorklistLimit     = 25
	maxListLimit             = 100
	defaultCashFlowPast      = 6
	defaultCashFlowFuture    = 7
	maxCashFlowWeeks         = 26
	worklistSeverityWatch    = "Watch"
	worklistSeverityWarning  = "Warning"
	worklistSeverityCritical = "Critical"
)

func (s *Service) GetDashboardKPIs(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDashboardKPIs, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)

	overview, err := s.analyticsRepo.GetBalanceOverview(ctx, req)
	if err != nil {
		return nil, err
	}

	series, err := s.analyticsRepo.ListBalanceSeries(ctx, customerledgerports.ListARSeriesRequest{
		TenantInfo: req.TenantInfo,
		AsOfDate:   req.AsOfDate,
		Weeks:      kpiDeltaWeeks,
	})
	if err != nil {
		return nil, err
	}

	totals, err := s.analyticsRepo.GetCollectionTotals(
		ctx,
		customerledgerports.GetARCollectionMetricsRequest{
			TenantInfo: req.TenantInfo,
			AsOfDate:   req.AsOfDate,
			PeriodDays: collectionPeriodDays,
		},
	)
	if err != nil {
		return nil, err
	}

	kpis := &customerledgerports.ARDashboardKPIs{
		AsOfDate:       req.AsOfDate,
		Overview:       overview,
		OverduePercent: percentOf(overview.OverdueMinor, overview.TotalOpenMinor),
		Collection:     collectionPerformance(totals),
	}
	if len(series) > 0 {
		last := series[len(series)-1]
		kpis.CurrentDSODays = computeDSODays(last.ARBalanceMinor, last.BilledMinor)
		kpis.DSODeltaDays = kpis.CurrentDSODays - computeDSODays(
			series[0].ARBalanceMinor,
			series[0].BilledMinor,
		)
	}
	return kpis, nil
}

func (s *Service) GetPaymentStats(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARPaymentStats, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	return s.analyticsRepo.GetPaymentStats(ctx, req)
}

func (s *Service) GetDraftInvoices(
	ctx context.Context,
	req customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDraftInvoices, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	return s.analyticsRepo.GetDraftInvoices(ctx, req)
}

func (s *Service) GetDSOTrend(
	ctx context.Context,
	req customerledgerports.ListARSeriesRequest,
) ([]*customerledgerports.ARDSOTrendPoint, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Weeks = clampWeeks(req.Weeks)

	series, err := s.analyticsRepo.ListBalanceSeries(ctx, req)
	if err != nil {
		return nil, err
	}

	points := make([]*customerledgerports.ARDSOTrendPoint, 0, len(series))
	for _, p := range series {
		points = append(points, &customerledgerports.ARDSOTrendPoint{
			PeriodEnd:      p.PeriodEnd,
			DSODays:        computeDSODays(p.ARBalanceMinor, p.BilledMinor),
			ARBalanceMinor: p.ARBalanceMinor,
			BilledMinor:    p.BilledMinor,
		})
	}
	return points, nil
}

func (s *Service) GetAgingTrend(
	ctx context.Context,
	req customerledgerports.ListARSeriesRequest,
) ([]*customerledgerports.ARAgingTrendPoint, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Weeks = clampWeeks(req.Weeks)
	return s.analyticsRepo.ListAgingTrend(ctx, req)
}

func (s *Service) GetCashFlowForecast(
	ctx context.Context,
	req customerledgerports.ListARCashFlowRequest,
) ([]*customerledgerports.ARCashFlowPoint, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	if req.PastWeeks <= 0 {
		req.PastWeeks = defaultCashFlowPast
	}
	if req.FutureWeeks <= 0 {
		req.FutureWeeks = defaultCashFlowFuture
	}
	req.PastWeeks = min(req.PastWeeks, maxCashFlowWeeks)
	req.FutureWeeks = min(req.FutureWeeks, maxCashFlowWeeks)
	return s.analyticsRepo.ListCashFlow(ctx, req)
}

func (s *Service) ListTopOverdueCustomers(
	ctx context.Context,
	req customerledgerports.ListARTopOverdueCustomersRequest,
) ([]*customerledgerports.ARTopOverdueCustomer, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Limit = clampLimit(req.Limit, defaultTopOverdueLimit)
	return s.analyticsRepo.ListTopOverdueCustomers(ctx, req)
}

func (s *Service) GetCollectionsWorklist(
	ctx context.Context,
	req customerledgerports.ListARCollectionsWorklistRequest,
) ([]*customerledgerports.ARCollectionsWorklistItem, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Limit = clampLimit(req.Limit, defaultWorklistLimit)

	items, err := s.analyticsRepo.ListCollectionsWorklist(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.Severity = worklistSeverity(item)
	}
	return items, nil
}

func (s *Service) GetCustomerProfile(
	ctx context.Context,
	req customerledgerports.GetARCustomerSnapshotRequest,
) (*customerledgerports.ARCustomerProfile, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)

	snapshot, err := s.analyticsRepo.GetCustomerSnapshot(ctx, req)
	if err != nil {
		return nil, err
	}

	profile := &customerledgerports.ARCustomerProfile{
		Snapshot: snapshot,
		DSODays:  computeDSODays(snapshot.TotalOpenMinor, snapshot.BilledTrailing91Minor),
	}
	if snapshot.HasCreditLimit && snapshot.CreditLimitMinor > 0 {
		profile.CreditUtilization = float64(snapshot.TotalOpenMinor) / float64(
			snapshot.CreditLimitMinor,
		)
	}
	profile.DelinquencyScore = computeDelinquencyScore(snapshot)
	return profile, nil
}

func normalizeAsOf(asOfDate int64) int64 {
	if asOfDate <= 0 {
		return timeutils.NowUnix()
	}
	return asOfDate
}

func clampWeeks(weeks int) int {
	if weeks <= 0 {
		return defaultTrendWeeks
	}
	return min(weeks, maxTrendWeeks)
}

func clampLimit(limit, fallback int) int {
	if limit <= 0 {
		return fallback
	}
	return min(limit, maxListLimit)
}

func computeDSODays(balanceMinor, billedMinor int64) float64 {
	if billedMinor <= 0 {
		return 0
	}
	return float64(balanceMinor) / float64(billedMinor) * dsoWindowDays
}

func computeCEI(totals *customerledgerports.ARCollectionTotals) float64 {
	if totals == nil {
		return 0
	}
	denominator := totals.BeginningOpenMinor + totals.CreditSalesMinor - totals.EndingCurrentMinor
	if denominator <= 0 {
		return 0
	}
	numerator := totals.BeginningOpenMinor + totals.CreditSalesMinor - totals.EndingOpenMinor
	cei := float64(numerator) / float64(denominator) * 100
	return math.Max(0, math.Min(cei, 100))
}

func collectionPerformance(
	totals *customerledgerports.ARCollectionTotals,
) *customerledgerports.ARCollectionPerformance {
	return &customerledgerports.ARCollectionPerformance{
		Totals:        totals,
		CEI:           computeCEI(totals),
		WriteOffRatio: ratioOf(totals.ShortPayMinor, totals.CreditSalesMinor),
		DisputeRate: ratioOf(
			int64(totals.DisputedInvoiceCount),
			int64(totals.PostedInvoiceCount),
		),
		ShortPayRate: ratioOf(
			int64(totals.ShortPayApplicationCount),
			int64(totals.ApplicationCount),
		),
	}
}

func percentOf(part, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func ratioOf(part, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func worklistSeverity(item *customerledgerports.ARCollectionsWorklistItem) string {
	switch {
	case item.DaysPastDue >= 30:
		return worklistSeverityCritical
	case item.DaysPastDue >= 15 || item.IsDisputed:
		return worklistSeverityWarning
	default:
		return worklistSeverityWatch
	}
}

func computeDelinquencyScore(snapshot *customerledgerports.ARCustomerSnapshot) float64 {
	if snapshot == nil || snapshot.TotalOpenMinor <= 0 {
		return 0
	}
	overdueShare := percentOf(snapshot.OverdueMinor, snapshot.TotalOpenMinor)
	ageFactor := math.Min(float64(snapshot.OldestDaysPastDue)/90.0, 1) * 100
	payFactor := math.Min(math.Max(snapshot.AvgDaysToPay-30, 0)/60.0, 1) * 100
	score := overdueShare*0.5 + ageFactor*0.3 + payFactor*0.2
	return math.Max(0, math.Min(score, 100))
}
