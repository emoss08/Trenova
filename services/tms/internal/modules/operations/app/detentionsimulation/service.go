package detentionsimulation

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

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

	facts, err := s.analyticsRepo.ListStopFacts(ctx, &detentionports.ListStopFactsRequest{
		TenantInfo: tenantInfo,
		From:       req.From,
		To:         req.To,
		Limit:      limit + 1,
	})
	if err != nil {
		return nil, err
	}

	result := &BacktestResult{
		ProposedRevenue:     decimal.Zero,
		BaselineRevenue:     decimal.Zero,
		RevenueDelta:        decimal.Zero,
		ProposedDriverPay:   decimal.NewNullDecimal(decimal.Zero),
		ProposedNetMargin:   decimal.NewNullDecimal(decimal.Zero),
		NegativeMarginStops: new(int),
		From:                req.From,
		To:                  req.To,
	}

	if len(facts) > limit {
		result.Truncated = true
		facts = facts[:limit]
	}

	accessorial, err := s.accessorialRepo.GetByID(
		ctx,
		accessorialchargeports.GetAccessorialChargeByIDRequest{
			ID:         req.Policy.AccessorialChargeID,
			TenantInfo: &tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	req.Policy.SpecificityScore = req.Policy.ComputeSpecificity()
	payRate := parsePayRate(req.DriverPayRate)
	location, err := s.effects.TenantLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

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

		if !req.Policy.Matches(detentionpolicy.MatchInput{
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

		snapshot := detentionpolicy.NewPolicySnapshot(detentionpolicy.SnapshotInput{
			Policy:          req.Policy,
			StopType:        fact.StopType,
			FreeMinutes:     req.Policy.FreeMinutesForStopType(stopType),
			AccessorialCode: accessorial.Code,
			FlatRate:        accessorial.Amount,
			FlatRateUnit:    detentionengine.TierUnitFromAccessorial(accessorial.RateUnit),
			CapturedAt:      *fact.ActualArrival,
		})

		computed := detentionengine.Compute(detentionengine.ComputeInput{
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
			req.Policy.LateArrivalRule == detentionpolicy.LateArrivalRuleForfeit {
			result.StopsForfeited++
		}
		if computed.SuppressedByGate {
			result.StopsSuppressed++
		}
		if computed.BillableAmount.IsPositive() {
			result.StopsBillable++
		}
		if computed.NetMargin.IsNegative() {
			*result.NegativeMarginStops++
		}

		prior := baseline[fact.StopID.String()]

		result.ProposedRevenue = result.ProposedRevenue.Add(computed.BillableAmount)
		result.BaselineRevenue = result.BaselineRevenue.Add(prior)
		result.ProposedDriverPay = addNull(result.ProposedDriverPay, computed.DriverPayAmount)
		result.ProposedNetMargin = addNull(result.ProposedNetMargin, computed.NetMargin)

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

	names, err := s.analyticsRepo.EntityNames(ctx, &detentionports.DetentionEntityNamesRequest{
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
	rows, err := s.analyticsRepo.BaselineByStop(ctx, &detentionports.DetentionStatsRequest{
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

// FacilityProfiles ranks facilities by what their docks actually cost. The p90
// dwell is the number that matters at planning time: an average hides the
// Friday afternoons that blow through free time.
func (s *Service) FacilityProfiles(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.FacilityDetentionStat, error) {
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
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.CustomerDetentionStat, error) {
	if s.analyticsRepo == nil {
		return nil, errortypes.NewValidationError(
			"analytics", errortypes.ErrSystemError, "Detention analytics are not available")
	}
	return s.analyticsRepo.CustomerStats(ctx, req)
}

// WaiverLeakage reports where discretionary revenue went, by coded reason.
func (s *Service) WaiverLeakage(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.WaiverLeakageStat, error) {
	if s.analyticsRepo == nil {
		return nil, errortypes.NewValidationError(
			"analytics", errortypes.ErrSystemError, "Detention analytics are not available")
	}
	return s.analyticsRepo.WaiverStats(ctx, req)
}

type Service struct {
	accessorialRepo accessorialchargeports.AccessorialChargeRepository
	analyticsRepo   detentionports.DetentionAnalyticsRepository
	now             func() int64
	effects         *detentioneffects.Effects
}

type Deps struct {
	AccessorialRepo accessorialchargeports.AccessorialChargeRepository
	AnalyticsRepo   detentionports.DetentionAnalyticsRepository
	Now             func() int64
	Effects         *detentioneffects.Effects
}

func New(d Deps) *Service {
	return &Service{
		accessorialRepo: d.AccessorialRepo,
		analyticsRepo:   d.AnalyticsRepo,
		now:             d.Now,
		effects:         d.Effects,
	}
}

// PreviewPolicy runs the real engine against a hypothetical stop.
//
// The preview deliberately reuses the production calculator rather than
// re-deriving the math for the UI. Configuration without an accurate feedback
// loop is how carriers end up with the invoice error rate the industry has.
func (s *Service) PreviewPolicy(
	ctx context.Context,
	policy *detentionpolicy.DetentionPolicy,
	scenario PreviewScenario,
	tenantInfo pagination.TenantInfo,
) (*PreviewResult, error) {
	if policy == nil {
		return nil, errortypes.NewValidationError(
			"policy", errortypes.ErrRequired, "A policy is required to preview")
	}

	multiErr := errortypes.NewMultiError()
	policy.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if scenario.ArrivedAt <= 0 {
		return nil, errortypes.NewValidationError(
			"scenario.arrivedAt", errortypes.ErrRequired, "An arrival time is required")
	}

	if scenario.StopType == "" {
		scenario.StopType = shipment.StopTypeDelivery
	}
	if scenario.ScheduleType == "" {
		scenario.ScheduleType = shipment.StopScheduleTypeAppointment
	}

	accessorial, err := s.accessorialRepo.GetByID(
		ctx,
		accessorialchargeports.GetAccessorialChargeByIDRequest{
			ID:         policy.AccessorialChargeID,
			TenantInfo: &tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	policy.SpecificityScore = policy.ComputeSpecificity()

	now := s.now()
	if scenario.DepartedAt != nil && *scenario.DepartedAt > now {
		now = *scenario.DepartedAt
	}

	snapshot := detentionpolicy.NewPolicySnapshot(detentionpolicy.SnapshotInput{
		Policy:          policy,
		StopType:        string(scenario.StopType),
		FreeMinutes:     policy.FreeMinutesForStopType(scenario.StopType),
		AccessorialCode: accessorial.Code,
		FlatRate:        accessorial.Amount,
		FlatRateUnit:    detentionengine.TierUnitFromAccessorial(accessorial.RateUnit),
		CapturedAt:      now,
	})

	location, err := s.effects.TenantLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	computed := detentionengine.Compute(detentionengine.ComputeInput{
		Snapshot:         snapshot,
		StopType:         scenario.StopType,
		ScheduleType:     scenario.ScheduleType,
		ArrivedAt:        &scenario.ArrivedAt,
		DepartedAt:       scenario.DepartedAt,
		AppointmentStart: scenario.AppointmentStart,
		AppointmentEnd:   scenario.AppointmentEnd,
		NoticeSentAt:     scenario.NoticeSentAt,
		Now:              now,
		Location:         location,
		DriverPayRate:    parsePayRate(scenario.DriverPayRate),
		ShipmentAccrued:  decimal.Zero,
	})

	result := &PreviewResult{
		Snapshot:           snapshot,
		RawDwellMinutes:    computed.RawDwellMinutes,
		FreeMinutesGranted: computed.FreeMinutesGranted,
		BillableMinutes:    computed.BillableMinutes,
		RoundedMinutes:     computed.RoundedMinutes,
		BillableAmount:     computed.BillableAmount,
		GrossAmount:        computed.GrossAmount,
		DriverPayAmount:    decimal.NewNullDecimal(computed.DriverPayAmount),
		NetMargin:          decimal.NewNullDecimal(computed.NetMargin),
		ArrivedLate:        computed.ArrivedLate,
		CapApplied:         computed.CapApplied,
		Status:             computed.Status,
		NotificationStatus: computed.NotificationStatus,
		SuppressedByGate:   computed.SuppressedByGate,
		Trace:              computed.Trace,
	}

	if computed.Trace != nil {
		result.Receipt = computed.Trace.Receipt()
	}

	return result, nil
}
