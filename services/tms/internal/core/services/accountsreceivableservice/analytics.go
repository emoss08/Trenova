package accountsreceivableservice

import (
	"context"
	"math"

	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	defaultTrendWeeks        = 13
	maxTrendWeeks            = 52
	dsoWindowDays            = 91.0
	collectionPeriodDays     = 91
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
	req repositoryports.GetARAnalyticsRequest,
) (*serviceports.ARDashboardKPIs, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)

	overview, err := s.analyticsRepo.GetBalanceOverview(ctx, req)
	if err != nil {
		return nil, err
	}

	series, err := s.analyticsRepo.ListBalanceSeries(ctx, repositoryports.ListARSeriesRequest{
		TenantInfo: req.TenantInfo,
		AsOfDate:   req.AsOfDate,
		Weeks:      kpiDeltaWeeks,
	})
	if err != nil {
		return nil, err
	}

	totals, err := s.analyticsRepo.GetCollectionTotals(
		ctx,
		repositoryports.GetARCollectionMetricsRequest{
			TenantInfo: req.TenantInfo,
			AsOfDate:   req.AsOfDate,
			PeriodDays: collectionPeriodDays,
		},
	)
	if err != nil {
		return nil, err
	}

	kpis := &serviceports.ARDashboardKPIs{
		AsOfDate:       req.AsOfDate,
		Overview:       overview,
		CEI:            computeCEI(totals),
		AvgDaysToPay:   totals.AvgDaysToPay,
		OverduePercent: percentOf(overview.OverdueMinor, overview.TotalOpenMinor),
		WriteOffRatio:  ratioOf(totals.ShortPayMinor, totals.CreditSalesMinor),
		DisputeRate: ratioOf(
			int64(totals.DisputedInvoiceCount),
			int64(totals.PostedInvoiceCount),
		),
		ShortPayRate: ratioOf(
			int64(totals.ShortPayApplicationCount),
			int64(totals.ApplicationCount),
		),
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
	req repositoryports.GetARAnalyticsRequest,
) (*repositoryports.ARPaymentStats, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	return s.analyticsRepo.GetPaymentStats(ctx, req)
}

func (s *Service) GetDSOTrend(
	ctx context.Context,
	req repositoryports.ListARSeriesRequest,
) ([]*serviceports.ARDSOTrendPoint, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Weeks = clampWeeks(req.Weeks)

	series, err := s.analyticsRepo.ListBalanceSeries(ctx, req)
	if err != nil {
		return nil, err
	}

	points := make([]*serviceports.ARDSOTrendPoint, 0, len(series))
	for _, p := range series {
		points = append(points, &serviceports.ARDSOTrendPoint{
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
	req repositoryports.ListARSeriesRequest,
) ([]*repositoryports.ARAgingTrendPoint, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Weeks = clampWeeks(req.Weeks)
	return s.analyticsRepo.ListAgingTrend(ctx, req)
}

func (s *Service) GetCashFlowForecast(
	ctx context.Context,
	req repositoryports.ListARCashFlowRequest,
) ([]*repositoryports.ARCashFlowPoint, error) {
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

func (s *Service) GetCollectionPerformance(
	ctx context.Context,
	req repositoryports.GetARCollectionMetricsRequest,
) (*serviceports.ARCollectionPerformance, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	if req.PeriodDays <= 0 {
		req.PeriodDays = collectionPeriodDays
	}

	totals, err := s.analyticsRepo.GetCollectionTotals(ctx, req)
	if err != nil {
		return nil, err
	}

	return &serviceports.ARCollectionPerformance{
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
	}, nil
}

func (s *Service) ListTopOverdueCustomers(
	ctx context.Context,
	req repositoryports.ListARTopOverdueCustomersRequest,
) ([]*repositoryports.ARTopOverdueCustomer, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)
	req.Limit = clampLimit(req.Limit, defaultTopOverdueLimit)
	return s.analyticsRepo.ListTopOverdueCustomers(ctx, req)
}

func (s *Service) GetCollectionsWorklist(
	ctx context.Context,
	req repositoryports.ListARCollectionsWorklistRequest,
) ([]*repositoryports.ARCollectionsWorklistItem, error) {
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
	req repositoryports.GetARCustomerSnapshotRequest,
) (*serviceports.ARCustomerProfile, error) {
	req.AsOfDate = normalizeAsOf(req.AsOfDate)

	snapshot, err := s.analyticsRepo.GetCustomerSnapshot(ctx, req)
	if err != nil {
		return nil, err
	}

	profile := &serviceports.ARCustomerProfile{
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

func computeCEI(totals *repositoryports.ARCollectionTotals) float64 {
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

func worklistSeverity(item *repositoryports.ARCollectionsWorklistItem) string {
	switch {
	case item.DaysPastDue >= 30:
		return worklistSeverityCritical
	case item.DaysPastDue >= 15 || item.IsDisputed:
		return worklistSeverityWarning
	default:
		return worklistSeverityWatch
	}
}

func computeDelinquencyScore(snapshot *repositoryports.ARCustomerSnapshot) float64 {
	if snapshot == nil || snapshot.TotalOpenMinor <= 0 {
		return 0
	}
	overdueShare := percentOf(snapshot.OverdueMinor, snapshot.TotalOpenMinor)
	ageFactor := math.Min(float64(snapshot.OldestDaysPastDue)/90.0, 1) * 100
	payFactor := math.Min(math.Max(snapshot.AvgDaysToPay-30, 0)/60.0, 1) * 100
	score := overdueShare*0.5 + ageFactor*0.3 + payFactor*0.2
	return math.Max(0, math.Min(score, 100))
}
