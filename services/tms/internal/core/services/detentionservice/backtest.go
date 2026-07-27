package detentionservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// BacktestRequest re-prices settled history under proposed terms.
type BacktestRequest struct {
	Policy        *detention.DetentionPolicy `json:"policy"`
	CompareToID   *pulid.ID                  `json:"compareToPolicyId"`
	From          int64                      `json:"from"`
	To            int64                      `json:"to"`
	Limit         int                        `json:"limit"`
	DriverPayRate *string                    `json:"driverPayRate"`

	// AssumeNoticeCompliance models the notice as having been sent on time.
	// Defaults to true, because a backtest is asking what the *terms* would
	// earn, not re-litigating past process failures. Set false to see what the
	// same history would have produced given the notices actually sent.
	AssumeNoticeCompliance *bool `json:"assumeNoticeCompliance"`
}

// BacktestBucket is one grouped slice of the backtest result.
type BacktestBucket struct {
	Key             string          `json:"key"`
	Label           string          `json:"label"`
	StopCount       int             `json:"stopCount"`
	BillableCount   int             `json:"billableCount"`
	ProposedAmount  decimal.Decimal `json:"proposedAmount"`
	BaselineAmount  decimal.Decimal `json:"baselineAmount"`
	Delta           decimal.Decimal `json:"delta"`
	DriverPayAmount decimal.Decimal `json:"driverPayAmount"`
	NetMargin       decimal.Decimal `json:"netMargin"`
}

// BacktestResult is the projected impact of a policy change.
type BacktestResult struct {
	StopsEvaluated      int              `json:"stopsEvaluated"`
	StopsMatched        int              `json:"stopsMatched"`
	StopsBillable       int              `json:"stopsBillable"`
	StopsForfeited      int              `json:"stopsForfeited"`
	StopsSuppressed     int              `json:"stopsSuppressed"`
	ProposedRevenue     decimal.Decimal  `json:"proposedRevenue"`
	BaselineRevenue     decimal.Decimal  `json:"baselineRevenue"`
	RevenueDelta        decimal.Decimal  `json:"revenueDelta"`
	ProposedDriverPay   decimal.Decimal  `json:"proposedDriverPay"`
	ProposedNetMargin   decimal.Decimal  `json:"proposedNetMargin"`
	NegativeMarginStops int              `json:"negativeMarginStops"`
	ByCustomer          []BacktestBucket `json:"byCustomer"`
	ByFacility          []BacktestBucket `json:"byFacility"`
	From                int64            `json:"from"`
	To                  int64            `json:"to"`
	Truncated           bool             `json:"truncated"`
}

const defaultBacktestLimit = 2000

// Backtest replays proposed policy terms against settled dwell history.
//
// Because occurrences are computed from immutable inputs by a deterministic
// engine, re-pricing the past is exact rather than approximate: the same stop
// facts run through the same calculator, only with different terms. That makes
// "what happens if we cut free time to 90 minutes" answerable with a number
// before the contract is signed.
func (s *Service) Backtest(
	ctx context.Context,
	req BacktestRequest,
	tenantInfo pagination.TenantInfo,
) (*BacktestResult, error) {
	if req.Policy == nil {
		return nil, errortypes.NewValidationError(
			"policy", errortypes.ErrRequired, "A policy is required to backtest")
	}

	multiErr := errortypes.NewMultiError()
	req.Policy.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if s.analyticsRepo == nil {
		return nil, errortypes.NewValidationError(
			"policy", errortypes.ErrSystemError, "Backtesting is not available")
	}

	limit := req.Limit
	if limit <= 0 || limit > defaultBacktestLimit {
		limit = defaultBacktestLimit
	}

	facts, err := s.analyticsRepo.ListStopFacts(ctx, &repositories.ListStopFactsRequest{
		TenantInfo: tenantInfo,
		From:       req.From,
		To:         req.To,
		Limit:      limit + 1,
	})
	if err != nil {
		return nil, err
	}

	result := &BacktestResult{
		ProposedRevenue:   decimal.Zero,
		BaselineRevenue:   decimal.Zero,
		RevenueDelta:      decimal.Zero,
		ProposedDriverPay: decimal.Zero,
		ProposedNetMargin: decimal.Zero,
		From:              req.From,
		To:                req.To,
	}

	if len(facts) > limit {
		result.Truncated = true
		facts = facts[:limit]
	}

	accessorial, err := s.accessorialRepo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID:         req.Policy.AccessorialChargeID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	req.Policy.SpecificityScore = req.Policy.ComputeSpecificity()
	payRate := parsePayRate(req.DriverPayRate)
	location := s.tenantLocation(ctx, tenantInfo.OrgID)

	baseline, err := s.baselineByStop(ctx, tenantInfo, req.From, req.To)
	if err != nil {
		return nil, err
	}

	assumeNotice := req.AssumeNoticeCompliance == nil || *req.AssumeNoticeCompliance

	byCustomer := map[string]*BacktestBucket{}
	byFacility := map[string]*BacktestBucket{}

	for _, fact := range facts {
		if fact == nil || fact.ActualArrival == nil {
			continue
		}

		result.StopsEvaluated++

		if fact.CountDetentionOverride != nil && !*fact.CountDetentionOverride {
			continue
		}

		stopType := shipment.StopType(fact.StopType)
		scheduleType := shipment.StopScheduleType(fact.ScheduleType)

		if !req.Policy.Matches(detention.MatchInput{
			CustomerID:     fact.CustomerID,
			LocationID:     fact.LocationID,
			ShipmentTypeID: fact.ShipmentTypeID,
			ServiceTypeID:  fact.ServiceTypeID,
			StopType:       stopType,
			ScheduleType:   scheduleType,
			Timestamp:      *fact.ActualArrival,
		}) {
			continue
		}

		result.StopsMatched++

		snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
			Policy:          req.Policy,
			StopType:        fact.StopType,
			FreeMinutes:     req.Policy.FreeMinutesForStopType(stopType),
			AccessorialCode: accessorial.Code,
			FlatRate:        accessorial.Amount,
			FlatRateUnit:    tierUnitFromAccessorial(accessorial.RateUnit),
			CapturedAt:      *fact.ActualArrival,
		})

		computed := Compute(ComputeInput{
			Snapshot:         snapshot,
			StopType:         stopType,
			ScheduleType:     scheduleType,
			ArrivedAt:        fact.ActualArrival,
			DepartedAt:       fact.ActualDeparture,
			AppointmentStart: appointmentStartFromFact(fact),
			AppointmentEnd:   fact.ScheduledWindowEnd,
			NoticeSentAt:     noticeSentFor(assumeNotice, fact.ActualArrival),
			Now:              derefOr(fact.ActualDeparture, *fact.ActualArrival),
			Location:         location,
			DriverPayRate:    payRate,
			ShipmentAccrued:  decimal.Zero,
		})

		if computed.ArrivedLate &&
			req.Policy.LateArrivalRule == detention.LateArrivalRuleForfeit {
			result.StopsForfeited++
		}
		if computed.SuppressedByGate {
			result.StopsSuppressed++
		}
		if computed.BillableAmount.IsPositive() {
			result.StopsBillable++
		}
		if computed.NetMargin.IsNegative() {
			result.NegativeMarginStops++
		}

		prior := baseline[fact.StopID.String()]

		result.ProposedRevenue = result.ProposedRevenue.Add(computed.BillableAmount)
		result.BaselineRevenue = result.BaselineRevenue.Add(prior)
		result.ProposedDriverPay = result.ProposedDriverPay.Add(computed.DriverPayAmount)
		result.ProposedNetMargin = result.ProposedNetMargin.Add(computed.NetMargin)

		accumulate(byCustomer, fact.CustomerID.String(), computed, prior)
		accumulate(byFacility, fact.LocationID.String(), computed, prior)
	}

	result.RevenueDelta = result.ProposedRevenue.Sub(result.BaselineRevenue)
	result.ByCustomer = rankBuckets(byCustomer)
	result.ByFacility = rankBuckets(byFacility)

	if err = s.labelBuckets(ctx, tenantInfo, result); err != nil {
		return nil, err
	}

	return result, nil
}

// labelBuckets swaps raw IDs for display names so a backtest reads as
// "Acme Foods −$1,240", not a PULID. A bucket whose entity has been deleted
// keeps its key as the label.
func (s *Service) labelBuckets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	result *BacktestResult,
) error {
	if len(result.ByCustomer) == 0 && len(result.ByFacility) == 0 {
		return nil
	}

	customerIDs := make([]pulid.ID, 0, len(result.ByCustomer))
	for i := range result.ByCustomer {
		if id, err := pulid.MustParse(result.ByCustomer[i].Key); err == nil {
			customerIDs = append(customerIDs, id)
		}
	}

	locationIDs := make([]pulid.ID, 0, len(result.ByFacility))
	for i := range result.ByFacility {
		if id, err := pulid.MustParse(result.ByFacility[i].Key); err == nil {
			locationIDs = append(locationIDs, id)
		}
	}

	names, err := s.analyticsRepo.EntityNames(ctx, &repositories.DetentionEntityNamesRequest{
		TenantInfo:  tenantInfo,
		LocationIDs: locationIDs,
		CustomerIDs: customerIDs,
	})
	if err != nil {
		return err
	}

	for i := range result.ByCustomer {
		bucket := &result.ByCustomer[i]
		bucket.Label = bucket.Key
		if id, pErr := pulid.MustParse(bucket.Key); pErr == nil {
			if name, ok := names.Customers[id]; ok && name != "" {
				bucket.Label = name
			}
		}
	}

	for i := range result.ByFacility {
		bucket := &result.ByFacility[i]
		bucket.Label = bucket.Key
		if id, pErr := pulid.MustParse(bucket.Key); pErr == nil {
			if name, ok := names.Locations[id]; ok && name != "" {
				bucket.Label = name
			}
		}
	}

	return nil
}

// baselineByStop maps each stop to what it actually billed, so the backtest
// reports a delta against reality rather than against zero.
func (s *Service) baselineByStop(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	from, to int64,
) (map[string]decimal.Decimal, error) {
	rows, err := s.analyticsRepo.BaselineByStop(ctx, &repositories.DetentionStatsRequest{
		TenantInfo: tenantInfo,
		From:       from,
		To:         to,
	})
	if err != nil {
		return nil, err
	}

	baseline := make(map[string]decimal.Decimal, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		baseline[row.StopID.String()] = row.BillableAmount
	}

	return baseline, nil
}

func accumulate(
	buckets map[string]*BacktestBucket,
	key string,
	computed ComputeResult,
	baseline decimal.Decimal,
) {
	bucket, ok := buckets[key]
	if !ok {
		bucket = &BacktestBucket{
			Key:             key,
			ProposedAmount:  decimal.Zero,
			BaselineAmount:  decimal.Zero,
			Delta:           decimal.Zero,
			DriverPayAmount: decimal.Zero,
			NetMargin:       decimal.Zero,
		}
		buckets[key] = bucket
	}

	bucket.StopCount++
	if computed.BillableAmount.IsPositive() {
		bucket.BillableCount++
	}

	bucket.ProposedAmount = bucket.ProposedAmount.Add(computed.BillableAmount)
	bucket.BaselineAmount = bucket.BaselineAmount.Add(baseline)
	bucket.Delta = bucket.ProposedAmount.Sub(bucket.BaselineAmount)
	bucket.DriverPayAmount = bucket.DriverPayAmount.Add(computed.DriverPayAmount)
	bucket.NetMargin = bucket.NetMargin.Add(computed.NetMargin)
}

// rankBuckets orders by the magnitude of the change, so the biggest movers in
// either direction surface first.
func rankBuckets(buckets map[string]*BacktestBucket) []BacktestBucket {
	ranked := make([]BacktestBucket, 0, len(buckets))
	for _, bucket := range buckets {
		ranked = append(ranked, *bucket)
	}

	sort.SliceStable(ranked, func(a, b int) bool {
		return ranked[a].Delta.Abs().GreaterThan(ranked[b].Delta.Abs())
	})

	return ranked
}

// noticeSentFor models notice compliance for a replayed stop.
func noticeSentFor(assumeCompliance bool, arrival *int64) *int64 {
	if !assumeCompliance {
		return nil
	}
	return arrival
}

func appointmentStartFromFact(fact *repositories.StopFact) *int64 {
	if fact.ScheduledWindowStart <= 0 {
		return nil
	}
	start := fact.ScheduledWindowStart
	return &start
}

func derefOr(value *int64, fallback int64) int64 {
	if value == nil {
		return fallback
	}
	return *value
}

// FacilityProfiles ranks facilities by what their docks actually cost. The p90
// dwell is the number that matters at planning time: an average hides the
// Friday afternoons that blow through free time.
func (s *Service) FacilityProfiles(
	ctx context.Context,
	req *repositories.DetentionStatsRequest,
) ([]*repositories.FacilityDetentionStat, error) {
	if s.analyticsRepo == nil {
		return nil, errortypes.NewValidationError(
			"analytics", errortypes.ErrSystemError, "Detention analytics are not available")
	}
	return s.analyticsRepo.FacilityStats(ctx, req)
}

// CustomerProfiles rolls detention up per customer, carrying both sides of the
// ledger so an unprofitable detention arrangement is visible.
func (s *Service) CustomerProfiles(
	ctx context.Context,
	req *repositories.DetentionStatsRequest,
) ([]*repositories.CustomerDetentionStat, error) {
	if s.analyticsRepo == nil {
		return nil, errortypes.NewValidationError(
			"analytics", errortypes.ErrSystemError, "Detention analytics are not available")
	}
	return s.analyticsRepo.CustomerStats(ctx, req)
}

// WaiverLeakage reports where discretionary revenue went, by coded reason.
func (s *Service) WaiverLeakage(
	ctx context.Context,
	req *repositories.DetentionStatsRequest,
) ([]*repositories.WaiverLeakageStat, error) {
	if s.analyticsRepo == nil {
		return nil, errortypes.NewValidationError(
			"analytics", errortypes.ErrSystemError, "Detention analytics are not available")
	}
	return s.analyticsRepo.WaiverStats(ctx, req)
}
