package detentionservice

import (
	"fmt"
	"github.com/emoss08/trenova/shared/intutils"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/shopspring/decimal"
)

const (
	secondsPerMinute = 60
	minutesPerHour   = 60
	minutesPerDay    = 1440
)

var (
	minutesPerHourDec = decimal.NewFromInt(minutesPerHour)
	minutesPerDayDec  = decimal.NewFromInt(minutesPerDay)
)

// ComputeInput is the complete, immutable set of facts a detention charge is
// derived from. Nothing outside this struct may influence the result, which is
// what makes a stored trace replayable months later during a dispute.
type ComputeInput struct {
	Snapshot         detention.PolicySnapshot
	StopType         shipment.StopType
	ScheduleType     shipment.StopScheduleType
	ArrivedAt        *int64
	DepartedAt       *int64
	AppointmentStart *int64
	AppointmentEnd   *int64
	NoticeSentAt     *int64
	Now              int64
	Location         *time.Location
	DriverPayRate    decimal.NullDecimal
	DayAccrued       map[string]decimal.Decimal
	ShipmentAccrued  decimal.Decimal
}

// ComputeResult is everything the engine derived, ready to be persisted onto an
// occurrence.
type ComputeResult struct {
	Applicable         bool
	ClockStartAt       int64
	ClockStopAt        *int64
	IsOpen             bool
	FreeTimeExpiresAt  int64
	NoticeDueAt        *int64
	NoticeDeadlineAt   *int64
	ArrivedLate        bool
	LateByMinutes      int32
	FreeMinutesGranted int32
	RawDwellMinutes    int32
	BillableMinutes    int32
	RoundedMinutes     int32
	BillableUnits      decimal.Decimal
	GrossAmount        decimal.Decimal
	BillableAmount     decimal.Decimal
	DriverPayMinutes   int32
	DriverPayAmount    decimal.Decimal
	NetMargin          decimal.Decimal
	CapApplied         detention.CapKind
	ConvertedToLayover bool
	Status             detention.OccurrenceStatus
	NotificationStatus detention.NotificationStatus
	SuppressedByGate   bool
	RequiresApproval   bool
	DayAllocation      map[string]decimal.Decimal
	Trace              *detention.CalculationTrace
}

// Compute derives a detention charge from immutable facts and frozen policy
// terms. It never reads the clock or any external state beyond ComputeInput.
func Compute(in ComputeInput) ComputeResult {
	snap := in.Snapshot
	trace := detention.NewTrace(in.Now)
	loc := in.Location
	if loc == nil {
		loc = time.UTC
	}

	result := ComputeResult{
		BillableUnits:      decimal.Zero,
		GrossAmount:        decimal.Zero,
		BillableAmount:     decimal.Zero,
		DriverPayAmount:    decimal.Zero,
		NetMargin:          decimal.Zero,
		CapApplied:         detention.CapKindNone,
		Status:             detention.OccurrenceStatusNotBillable,
		NotificationStatus: detention.NotificationStatusNotRequired,
		DayAllocation:      map[string]decimal.Decimal{},
		Trace:              trace,
	}

	if in.ArrivedAt == nil || *in.ArrivedAt <= 0 {
		return result
	}

	result.Applicable = true

	trace.Add(detention.TraceStep{
		Kind:   detention.TraceStepPolicyResolved,
		Label:  "Policy applied",
		Detail: fmt.Sprintf("%s (%s)", snap.PolicyName, snap.PolicyCode),
		Term:   fmt.Sprintf("specificity %d, priority %d", snap.SpecificityScore, snap.Priority),
	})

	clockStart, lateness := resolveClockStart(in, trace)
	result.ArrivedLate = lateness.isLate
	result.LateByMinutes = lateness.byMinutes

	freeMinutes := snap.FreeMinutes

	if lateness.isLate {
		switch snap.LateArrivalRule {
		case detention.LateArrivalRuleForfeit:
			trace.AddMinutes(detention.TraceStepLateArrival,
				"Detention forfeited",
				fmt.Sprintf("Arrived %d min after the appointment window", lateness.byMinutes),
				"Late arrival rule: Forfeit", lateness.byMinutes, true)
			result.ClockStartAt = clockStart
			result.FreeTimeExpiresAt = clockStart + int64(freeMinutes)*secondsPerMinute
			result.FreeMinutesGranted = freeMinutes
			result.IsOpen = in.DepartedAt == nil
			result.Status = detention.OccurrenceStatusNotBillable
			finalizeTrace(trace, result)
			return result
		case detention.LateArrivalRuleClockFromAppointment:
			if in.AppointmentStart != nil {
				clockStart = *in.AppointmentStart
				trace.AddTimestamp(detention.TraceStepLateArrival,
					"Clock anchored to appointment",
					"Lateness consumes the carrier's own free time",
					"Late arrival rule: ClockFromAppointment", clockStart)
			}
		case detention.LateArrivalRuleReduceFreeTime:
			reduced := freeMinutes - lateness.byMinutes
			if reduced < 0 {
				reduced = 0
			}
			trace.AddMinutes(detention.TraceStepLateArrival,
				"Free time reduced by lateness",
				fmt.Sprintf("%d min granted instead of %d", reduced, freeMinutes),
				"Late arrival rule: ReduceFreeTime", freeMinutes-reduced, true)
			freeMinutes = reduced
		case detention.LateArrivalRuleNoEffect:
			trace.AddMinutes(detention.TraceStepLateArrival,
				"Late arrival noted, entitlement unaffected",
				fmt.Sprintf("Arrived %d min late", lateness.byMinutes),
				"Late arrival rule: NoEffect", lateness.byMinutes, false)
		}
	}

	result.ClockStartAt = clockStart
	result.FreeMinutesGranted = freeMinutes
	result.FreeTimeExpiresAt = clockStart + int64(freeMinutes)*secondsPerMinute

	clockStop, isOpen := resolveClockStop(in, clockStart)
	result.IsOpen = isOpen
	if !isOpen {
		stop := clockStop
		result.ClockStopAt = &stop
	}

	trace.AddTimestamp(detention.TraceStepClockStop,
		clockStopLabel(isOpen), clockStopDetail(isOpen),
		"Clock stop basis: Departure", clockStop)

	rawSeconds := clockStop - clockStart
	if rawSeconds < 0 {
		rawSeconds = 0
	}
	result.RawDwellMinutes = int32(rawSeconds / secondsPerMinute)

	trace.AddMinutes(detention.TraceStepRawDwell, "Total time on site",
		formatMinutes(result.RawDwellMinutes), "", result.RawDwellMinutes, false)

	trace.AddMinutes(detention.TraceStepFreeTime, "Free time deducted",
		formatMinutes(freeMinutes), fmt.Sprintf("%d min contractual free time", freeMinutes),
		freeMinutes, true)

	billable := result.RawDwellMinutes - freeMinutes
	if billable < 0 {
		billable = 0
	}

	result.NotificationStatus, result.NoticeDueAt, result.NoticeDeadlineAt = resolveNotification(
		in, result.FreeTimeExpiresAt,
	)

	if billable == 0 {
		result.BillableMinutes = 0
		result.Status = openOrNotBillable(isOpen)
		result.DriverPayMinutes, result.DriverPayAmount = computeDriverPay(
			in,
			result.RawDwellMinutes,
			trace,
		)
		result.NetMargin = result.BillableAmount.Sub(result.DriverPayAmount)
		finalizeTrace(trace, result)
		return result
	}

	if snap.MinimumBillableMinutes > 0 && billable < snap.MinimumBillableMinutes {
		trace.AddMinutes(
			detention.TraceStepMinimum,
			"Below minimum billable threshold",
			fmt.Sprintf(
				"%d min is under the %d min minimum",
				billable,
				snap.MinimumBillableMinutes,
			),
			fmt.Sprintf("Minimum billable: %d min", snap.MinimumBillableMinutes),
			billable,
			true,
		)
		result.BillableMinutes = 0
		result.Status = openOrNotBillable(isOpen)
		result.DriverPayMinutes, result.DriverPayAmount = computeDriverPay(
			in,
			result.RawDwellMinutes,
			trace,
		)
		result.NetMargin = result.BillableAmount.Sub(result.DriverPayAmount)
		finalizeTrace(trace, result)
		return result
	}

	result.BillableMinutes = billable

	rounded := applyRounding(billable, snap.RoundingMode, snap.BillingIncrementMinutes)
	if rounded != billable {
		trace.AddMinutes(detention.TraceStepRounding,
			fmt.Sprintf("Rounded %s to %s", formatMinutes(billable), formatMinutes(rounded)),
			roundingDetail(snap.RoundingMode, snap.BillingIncrementMinutes),
			fmt.Sprintf("%s to %d min increments", snap.RoundingMode, snap.BillingIncrementMinutes),
			rounded, rounded < billable)
	}

	rounded, result.ConvertedToLayover, result.CapApplied = applyMinuteCaps(
		rounded, freeMinutes, snap, trace, result.CapApplied,
	)
	result.RoundedMinutes = rounded

	if rounded == 0 {
		result.Status = openOrNotBillable(isOpen)
		result.DriverPayMinutes, result.DriverPayAmount = computeDriverPay(
			in,
			result.RawDwellMinutes,
			trace,
		)
		result.NetMargin = result.BillableAmount.Sub(result.DriverPayAmount)
		finalizeTrace(trace, result)
		return result
	}

	units, gross := applyRate(rounded, snap, trace)
	result.BillableUnits = units
	result.GrossAmount = gross

	capped, capKind, allocation := applyAmountCaps(applyAmountCapsInput{
		gross:       gross,
		rounded:     rounded,
		snapshot:    snap,
		clockStart:  clockStart,
		freeMins:    freeMinutes,
		clockStop:   clockStop,
		location:    loc,
		dayAccrued:  in.DayAccrued,
		shipAccrued: in.ShipmentAccrued,
		trace:       trace,
	})
	result.BillableAmount = capped
	result.DayAllocation = allocation
	if capKind != detention.CapKindNone {
		result.CapApplied = capKind
	}

	result.DriverPayMinutes, result.DriverPayAmount = computeDriverPay(
		in,
		result.RawDwellMinutes,
		trace,
	)

	applyNotificationGate(&result, snap, trace)
	applyApprovalThresholds(&result, snap, trace)

	result.NetMargin = result.BillableAmount.Sub(result.DriverPayAmount)
	result.Status = resolveStatus(&result, snap, isOpen)

	finalizeTrace(trace, result)

	return result
}

type latenessResult struct {
	isLate    bool
	byMinutes int32
}

func resolveClockStart(in ComputeInput, trace *detention.CalculationTrace) (int64, latenessResult) {
	arrival := *in.ArrivedAt
	snap := in.Snapshot

	clockStart := arrival
	basisDetail := "Driver arrival"

	switch snap.ClockStartBasis {
	case detention.ClockStartBasisArrival:
		clockStart = arrival
		basisDetail = "Driver arrival"
	case detention.ClockStartBasisAppointment:
		if in.AppointmentStart != nil {
			clockStart = *in.AppointmentStart
			basisDetail = "Scheduled appointment, regardless of arrival"
		}
	case detention.ClockStartBasisLaterOfArrivalOrAppointment:
		if in.AppointmentStart != nil && *in.AppointmentStart > arrival {
			clockStart = *in.AppointmentStart
			basisDetail = "Appointment, because the driver arrived early"
		} else {
			basisDetail = "Driver arrival, at or after the appointment"
		}
	case detention.ClockStartBasisEarlierOfArrivalOrAppointment:
		if in.AppointmentStart != nil && *in.AppointmentStart < arrival {
			clockStart = *in.AppointmentStart
			basisDetail = "Appointment, because it preceded arrival"
		} else {
			basisDetail = "Driver arrival, because it preceded the appointment"
		}
	}

	trace.AddTimestamp(detention.TraceStepClockStart, "Clock started", basisDetail,
		"Clock start basis: "+snap.ClockStartBasis.String(), clockStart)

	return clockStart, detectLateness(in, arrival)
}

// detectLateness measures arrival against the close of the appointment window,
// falling back to the appointment start when no window end was scheduled.
func detectLateness(in ComputeInput, arrival int64) latenessResult {
	deadline := in.AppointmentEnd
	if deadline == nil {
		deadline = in.AppointmentStart
	}
	if deadline == nil {
		return latenessResult{}
	}

	grace := int64(in.Snapshot.LateArrivalGraceMinutes) * secondsPerMinute
	threshold := *deadline + grace

	if arrival <= threshold {
		return latenessResult{}
	}

	return latenessResult{
		isLate:    true,
		byMinutes: intutils.SafeToInt32((arrival - *deadline) / secondsPerMinute),
	}
}

func resolveClockStop(in ComputeInput, clockStart int64) (int64, bool) {
	if in.DepartedAt != nil && *in.DepartedAt > 0 {
		stop := *in.DepartedAt
		if stop < clockStart {
			stop = clockStart
		}
		return stop, false
	}

	projected := in.Now
	if projected < clockStart {
		projected = clockStart
	}

	return projected, true
}

// applyRounding collapses raw billable minutes onto the contract's increment.
// Rounding errors are a named cause of rejected detention invoices, so the mode
// and increment are always contract terms rather than engine defaults.
func applyRounding(minutes int32, mode detention.RoundingMode, increment int16) int32 {
	if mode == detention.RoundingModeExact || increment <= 0 {
		return minutes
	}

	inc := int32(increment)
	whole := minutes / inc
	remainder := minutes % inc

	if remainder == 0 {
		return minutes
	}

	switch mode {
	case detention.RoundingModeUp:
		return (whole + 1) * inc
	case detention.RoundingModeDown:
		return whole * inc
	case detention.RoundingModeNearest:
		if remainder*2 >= inc {
			return (whole + 1) * inc
		}
		return whole * inc
	case detention.RoundingModeExact:
		return minutes
	default:
		return minutes
	}
}

func applyMinuteCaps(
	rounded, freeMinutes int32,
	snap detention.PolicySnapshot,
	trace *detention.CalculationTrace,
	current detention.CapKind,
) (int32, bool, detention.CapKind) {
	capKind := current
	convertedToLayover := false

	if snap.ConvertToLayoverAtMinutes != nil {
		boundary := *snap.ConvertToLayoverAtMinutes - freeMinutes
		if boundary < 0 {
			boundary = 0
		}
		if rounded > boundary {
			trace.AddMinutes(detention.TraceStepLayover, "Converted to layover",
				fmt.Sprintf("Detention capped at %s; the remainder bills as layover",
					formatMinutes(boundary)),
				fmt.Sprintf("Layover conversion at %d min on site",
					*snap.ConvertToLayoverAtMinutes),
				boundary, true)
			rounded = boundary
			convertedToLayover = true
			capKind = detention.CapKindLayoverBoundary
		}
	}

	if snap.MaxBillableMinutesPerStop != nil && rounded > *snap.MaxBillableMinutesPerStop {
		trace.AddMinutes(detention.TraceStepMinutesCap, "Billable minutes capped",
			fmt.Sprintf("Reduced to %s", formatMinutes(*snap.MaxBillableMinutesPerStop)),
			fmt.Sprintf("Maximum billable: %d min per stop", *snap.MaxBillableMinutesPerStop),
			*snap.MaxBillableMinutesPerStop, true)
		rounded = *snap.MaxBillableMinutesPerStop
		capKind = detention.CapKindMaxMinutes
	}

	return rounded, convertedToLayover, capKind
}

// applyRate walks the graduated ladder when the contract has one, otherwise
// multiplies the flat accessorial rate.
func applyRate(
	rounded int32,
	snap detention.PolicySnapshot,
	trace *detention.CalculationTrace,
) (decimal.Decimal, decimal.Decimal) {
	if snap.RateSource == detention.RateSourceTiers && len(snap.Tiers) > 0 {
		return applyTieredRate(rounded, snap, trace)
	}

	units := unitsForMinutes(rounded, snap.FlatRateUnit)
	amount := snap.FlatRate.Mul(units).Round(2)

	trace.AddAmount(detention.TraceStepFlatRate,
		fmt.Sprintf("%s × %s", units.StringFixed(2), rateLabel(snap.FlatRate, snap.FlatRateUnit)),
		formatMinutes(rounded)+" billable",
		"Accessorial "+snap.AccessorialCode, amount, false)

	return units, amount
}

func applyTieredRate(
	rounded int32,
	snap detention.PolicySnapshot,
	trace *detention.CalculationTrace,
) (decimal.Decimal, decimal.Decimal) {
	tiers := make([]detention.TierSnapshot, len(snap.Tiers))
	copy(tiers, snap.Tiers)
	sort.SliceStable(
		tiers,
		func(a, b int) bool { return tiers[a].FromMinute < tiers[b].FromMinute },
	)

	totalUnits := decimal.Zero
	totalAmount := decimal.Zero

	for _, tier := range tiers {
		span := tier.SpanMinutes(rounded)
		if span <= 0 {
			continue
		}

		units := unitsForMinutes(span, tier.RateUnit)
		amount := tier.Rate.Mul(units).Round(2)

		totalUnits = totalUnits.Add(units)
		totalAmount = totalAmount.Add(amount)

		trace.AddAmount(detention.TraceStepTier,
			tierLabel(tier),
			fmt.Sprintf("%s at %s", formatMinutes(span), rateLabel(tier.Rate, tier.RateUnit)),
			tierBandTerm(tier), amount, false)
	}

	return totalUnits, totalAmount
}

func unitsForMinutes(minutes int32, unit detention.TierRateUnit) decimal.Decimal {
	if minutes <= 0 {
		return decimal.Zero
	}

	switch unit {
	case detention.TierRateUnitHour:
		return decimal.NewFromInt(int64(minutes)).Div(minutesPerHourDec).Round(4)
	case detention.TierRateUnitDay:
		return decimal.NewFromInt(int64(minutes)).Div(minutesPerDayDec).Round(4)
	case detention.TierRateUnitFlat:
		return decimal.NewFromInt(1)
	default:
		return decimal.NewFromInt(int64(minutes)).Div(minutesPerHourDec).Round(4)
	}
}

type applyAmountCapsInput struct {
	gross       decimal.Decimal
	rounded     int32
	snapshot    detention.PolicySnapshot
	clockStart  int64
	freeMins    int32
	clockStop   int64
	location    *time.Location
	dayAccrued  map[string]decimal.Decimal
	shipAccrued decimal.Decimal
	trace       *detention.CalculationTrace
}

// applyAmountCaps clamps the computed charge against the contract's ceilings.
// The per-day cap is the subtle one: a stay that spans midnight would otherwise
// generate two full daily charges, so the amount is allocated across calendar
// days in the organization's timezone and each day is capped independently.
func applyAmountCaps(
	in applyAmountCapsInput,
) (decimal.Decimal, detention.CapKind, map[string]decimal.Decimal) {
	amount := in.gross
	capKind := detention.CapKindNone
	snap := in.snapshot

	allocation := allocateByDay(allocateByDayInput{
		amount:    amount,
		rounded:   in.rounded,
		billStart: in.clockStart + int64(in.freeMins)*secondsPerMinute,
		clockStop: in.clockStop,
		location:  in.location,
	})

	if snap.MaxChargePerStop.Valid && amount.GreaterThan(snap.MaxChargePerStop.Decimal) {
		in.trace.AddAmount(detention.TraceStepAmountCap, "Per-stop cap applied",
			fmt.Sprintf("Reduced from %s", amount.StringFixed(2)),
			"Maximum charge per stop: "+snap.MaxChargePerStop.Decimal.StringFixed(2),
			snap.MaxChargePerStop.Decimal, true)
		amount = snap.MaxChargePerStop.Decimal
		capKind = detention.CapKindPerStopAmount
		allocation = rescaleAllocation(allocation, in.gross, amount)
	}

	if snap.MaxChargePerDay.Valid && len(allocation) > 0 {
		capped, changed := applyDailyCap(allocation, in.dayAccrued, snap.MaxChargePerDay.Decimal)
		if changed {
			in.trace.AddAmount(detention.TraceStepAmountCap, "Daily cap applied",
				fmt.Sprintf("Reduced from %s across %d calendar day(s)",
					amount.StringFixed(2), len(allocation)),
				"Maximum charge per day: "+snap.MaxChargePerDay.Decimal.StringFixed(2),
				capped, true)
			amount = capped
			capKind = detention.CapKindPerDayAmount
		}
	}

	if snap.MaxChargePerShipment.Valid {
		remaining := snap.MaxChargePerShipment.Decimal.Sub(in.shipAccrued)
		if remaining.LessThan(decimal.Zero) {
			remaining = decimal.Zero
		}
		if amount.GreaterThan(remaining) {
			in.trace.AddAmount(detention.TraceStepAmountCap, "Shipment cap applied",
				fmt.Sprintf("Reduced from %s; %s already accrued on this shipment",
					amount.StringFixed(2), in.shipAccrued.StringFixed(2)),
				"Maximum charge per shipment: "+
					snap.MaxChargePerShipment.Decimal.StringFixed(2),
				remaining, true)
			amount = remaining
			capKind = detention.CapKindPerShipment
			allocation = rescaleAllocation(allocation, in.gross, amount)
		}
	}

	return amount.Round(2), capKind, allocation
}

type allocateByDayInput struct {
	amount    decimal.Decimal
	rounded   int32
	billStart int64
	clockStop int64
	location  *time.Location
}

// allocateByDay splits the charge across the calendar days the billable window
// actually touches, pro-rata by minutes.
func allocateByDay(in allocateByDayInput) map[string]decimal.Decimal {
	allocation := map[string]decimal.Decimal{}

	if in.rounded <= 0 || in.clockStop <= in.billStart {
		if in.rounded > 0 {
			key := time.Unix(in.billStart, 0).In(in.location).Format(time.DateOnly)
			allocation[key] = in.amount
		}
		return allocation
	}

	totalSeconds := in.clockStop - in.billStart
	minutesByDay := map[string]int64{}

	cursor := in.billStart
	for cursor < in.clockStop {
		dayStart := time.Unix(cursor, 0).In(in.location)
		nextMidnight := time.Date(
			dayStart.Year(), dayStart.Month(), dayStart.Day()+1,
			0, 0, 0, 0, in.location,
		).Unix()

		segmentEnd := nextMidnight
		if segmentEnd > in.clockStop {
			segmentEnd = in.clockStop
		}

		minutesByDay[dayStart.Format(time.DateOnly)] += segmentEnd - cursor
		cursor = segmentEnd
	}

	if totalSeconds <= 0 {
		return allocation
	}

	totalDec := decimal.NewFromInt(totalSeconds)
	assigned := decimal.Zero
	keys := make([]string, 0, len(minutesByDay))
	for key := range minutesByDay {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i == len(keys)-1 {
			allocation[key] = in.amount.Sub(assigned).Round(2)
			continue
		}
		share := in.amount.Mul(decimal.NewFromInt(minutesByDay[key])).Div(totalDec).Round(2)
		allocation[key] = share
		assigned = assigned.Add(share)
	}

	return allocation
}

func applyDailyCap(
	allocation map[string]decimal.Decimal,
	priorAccrued map[string]decimal.Decimal,
	dailyCap decimal.Decimal,
) (decimal.Decimal, bool) {
	total := decimal.Zero
	changed := false

	keys := make([]string, 0, len(allocation))
	for key := range allocation {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		share := allocation[key]
		prior := decimal.Zero
		if priorAccrued != nil {
			if existing, ok := priorAccrued[key]; ok {
				prior = existing
			}
		}

		remaining := dailyCap.Sub(prior)
		if remaining.LessThan(decimal.Zero) {
			remaining = decimal.Zero
		}

		if share.GreaterThan(remaining) {
			share = remaining
			changed = true
		}

		allocation[key] = share
		total = total.Add(share)
	}

	return total.Round(2), changed
}

func rescaleAllocation(
	allocation map[string]decimal.Decimal,
	from, to decimal.Decimal,
) map[string]decimal.Decimal {
	if len(allocation) == 0 || from.IsZero() {
		return allocation
	}

	rescaled := make(map[string]decimal.Decimal, len(allocation))
	keys := make([]string, 0, len(allocation))
	for key := range allocation {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	assigned := decimal.Zero
	for i, key := range keys {
		if i == len(keys)-1 {
			rescaled[key] = to.Sub(assigned).Round(2)
			continue
		}
		share := allocation[key].Mul(to).Div(from).Round(2)
		rescaled[key] = share
		assigned = assigned.Add(share)
	}

	return rescaled
}

// computeDriverPay derives the AP side from the same dwell. Carriers routinely
// lose margin when a customer negotiates longer free time than the driver's pay
// profile grants; surfacing both numbers on one record makes that visible.
func computeDriverPay(
	in ComputeInput,
	rawDwellMinutes int32,
	trace *detention.CalculationTrace,
) (int32, decimal.Decimal) {
	payFree := in.Snapshot.PayFreeMinutes
	payMinutes := rawDwellMinutes - payFree
	if payMinutes < 0 {
		payMinutes = 0
	}

	if payMinutes == 0 || !in.DriverPayRate.Valid {
		return payMinutes, decimal.Zero
	}

	hours := decimal.NewFromInt(int64(payMinutes)).Div(minutesPerHourDec).Round(4)
	amount := in.DriverPayRate.Decimal.Mul(hours).Round(2)

	trace.AddAmount(detention.TraceStepDriverPay, "Driver detention pay",
		fmt.Sprintf("%s beyond %s of pay free time",
			formatMinutes(payMinutes), formatMinutes(payFree)),
		"Driver pay free time: "+formatMinutes(payFree), amount, false)

	return payMinutes, amount
}

func resolveNotification(
	in ComputeInput,
	freeTimeExpiresAt int64,
) (detention.NotificationStatus, *int64, *int64) {
	snap := in.Snapshot

	if snap.NotificationRequirement == detention.NotificationRequirementNone {
		return detention.NotificationStatusNotRequired, nil, nil
	}

	dueAt := freeTimeExpiresAt - int64(snap.NotificationLeadMinutes)*secondsPerMinute
	deadlineAt := freeTimeExpiresAt + int64(snap.NotificationDeadlineMinutes)*secondsPerMinute

	if in.NoticeSentAt != nil {
		if *in.NoticeSentAt <= deadlineAt {
			return detention.NotificationStatusSent, &dueAt, &deadlineAt
		}
		return detention.NotificationStatusLate, &dueAt, &deadlineAt
	}

	if in.Now > deadlineAt {
		return detention.NotificationStatusMissed, &dueAt, &deadlineAt
	}

	return detention.NotificationStatusPending, &dueAt, &deadlineAt
}

// applyNotificationGate enforces the contractual requirement that the customer
// be told detention started. A missed notice is the single most common reason a
// otherwise-valid detention claim goes uncollected.
func applyNotificationGate(
	result *ComputeResult,
	snap detention.PolicySnapshot,
	trace *detention.CalculationTrace,
) {
	if snap.NotificationRequirement != detention.NotificationRequirementRequired {
		return
	}

	if result.NotificationStatus.SatisfiesRequirement() {
		trace.Add(detention.TraceStep{
			Kind:   detention.TraceStepNotification,
			Label:  "Notice requirement satisfied",
			Detail: "Customer was notified within the contractual window",
			Term:   "Notification: Required",
		})
		return
	}

	if result.IsOpen && result.NotificationStatus == detention.NotificationStatusPending {
		return
	}

	switch snap.UnnotifiedBehavior {
	case detention.UnnotifiedBehaviorSuppress:
		trace.AddAmount(
			detention.TraceStepNotification,
			"Charge suppressed: notice not sent in time",
			fmt.Sprintf("Contract requires notice; status is %s", result.NotificationStatus),
			"Unnotified behavior: Suppress",
			decimal.Zero,
			true,
		)
		result.BillableAmount = decimal.Zero
		result.SuppressedByGate = true
	case detention.UnnotifiedBehaviorFlag:
		trace.Add(detention.TraceStep{
			Kind:  detention.TraceStepNotification,
			Label: "Held for review: notice not sent in time",
			Detail: fmt.Sprintf(
				"Contract requires notice; status is %s",
				result.NotificationStatus,
			),
			Term: "Unnotified behavior: Flag",
		})
		result.RequiresApproval = true
	case detention.UnnotifiedBehaviorBill:
		trace.Add(detention.TraceStep{
			Kind:   detention.TraceStepNotification,
			Label:  "Notice missed; billing anyway",
			Detail: "Collectability is reduced without a notice receipt",
			Term:   "Unnotified behavior: Bill",
		})
	}
}

func applyApprovalThresholds(
	result *ComputeResult,
	snap detention.PolicySnapshot,
	trace *detention.CalculationTrace,
) {
	if snap.RequireApprovalOverAmount.Valid &&
		result.BillableAmount.GreaterThan(snap.RequireApprovalOverAmount.Decimal) {
		trace.Add(detention.TraceStep{
			Kind:  detention.TraceStepNotification,
			Label: "Approval required",
			Detail: fmt.Sprintf("%s exceeds the %s approval threshold",
				result.BillableAmount.StringFixed(2),
				snap.RequireApprovalOverAmount.Decimal.StringFixed(2)),
			Term: "Approval threshold",
		})
		result.RequiresApproval = true
	}
}

func resolveStatus(
	result *ComputeResult,
	snap detention.PolicySnapshot,
	isOpen bool,
) detention.OccurrenceStatus {
	if isOpen {
		return detention.OccurrenceStatusAccruing
	}

	if result.SuppressedByGate || result.BillableAmount.LessThanOrEqual(decimal.Zero) {
		return detention.OccurrenceStatusNotBillable
	}

	if result.RequiresApproval {
		return detention.OccurrenceStatusPending
	}

	if snap.AutoApproveUnderAmount.Valid &&
		result.BillableAmount.LessThanOrEqual(snap.AutoApproveUnderAmount.Decimal) {
		return detention.OccurrenceStatusApproved
	}

	return detention.OccurrenceStatusPending
}

func openOrNotBillable(isOpen bool) detention.OccurrenceStatus {
	if isOpen {
		return detention.OccurrenceStatusAccruing
	}
	return detention.OccurrenceStatusNotBillable
}

func finalizeTrace(trace *detention.CalculationTrace, result ComputeResult) {
	trace.AddAmount(detention.TraceStepFinal, "Billable detention",
		fmt.Sprintf("%s billable of %s on site",
			formatMinutes(result.RoundedMinutes), formatMinutes(result.RawDwellMinutes)),
		"", result.BillableAmount, false)
}

func formatMinutes(minutes int32) string {
	if minutes <= 0 {
		return "0 min"
	}
	if minutes < minutesPerHour {
		return fmt.Sprintf("%d min", minutes)
	}

	hours := minutes / minutesPerHour
	rem := minutes % minutesPerHour
	if rem == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, rem)
}

func rateLabel(rate decimal.Decimal, unit detention.TierRateUnit) string {
	switch unit {
	case detention.TierRateUnitHour:
		return rate.StringFixed(2) + "/hr"
	case detention.TierRateUnitDay:
		return rate.StringFixed(2) + "/day"
	case detention.TierRateUnitFlat:
		return rate.StringFixed(2) + " flat"
	default:
		return rate.StringFixed(2)
	}
}

func tierLabel(tier detention.TierSnapshot) string {
	if tier.Label != "" {
		return tier.Label
	}
	return "Tier " + tierBandTerm(tier)
}

func tierBandTerm(tier detention.TierSnapshot) string {
	if tier.ToMinute == nil {
		return fmt.Sprintf("%s and beyond", formatMinutes(tier.FromMinute))
	}
	return fmt.Sprintf("%s–%s", formatMinutes(tier.FromMinute), formatMinutes(*tier.ToMinute))
}

func clockStopLabel(isOpen bool) string {
	if isOpen {
		return "Clock still running"
	}
	return "Clock stopped"
}

func clockStopDetail(isOpen bool) string {
	if isOpen {
		return "Projected to now; the driver has not departed"
	}
	return "Driver departure"
}

func roundingDetail(mode detention.RoundingMode, increment int16) string {
	if mode == detention.RoundingModeExact || increment <= 0 {
		return "Billed to the exact minute"
	}
	return fmt.Sprintf("%s to the nearest %d-minute increment",
		roundingVerb(mode), increment)
}

func roundingVerb(mode detention.RoundingMode) string {
	switch mode {
	case detention.RoundingModeUp:
		return "Rounded up"
	case detention.RoundingModeDown:
		return "Rounded down"
	case detention.RoundingModeNearest:
		return "Rounded"
	case detention.RoundingModeExact:
		return "Exact"
	default:
		return "Rounded"
	}
}

// tierUnitFromAccessorial maps an accessorial's rate unit onto the tier unit
// vocabulary so a flat-rate policy and a tiered one share one code path.
func tierUnitFromAccessorial(unit accessorialcharge.RateUnit) detention.TierRateUnit {
	switch unit {
	case accessorialcharge.RateUnitHour:
		return detention.TierRateUnitHour
	case accessorialcharge.RateUnitDay:
		return detention.TierRateUnitDay
	case accessorialcharge.RateUnitStop, accessorialcharge.RateUnitMile:
		return detention.TierRateUnitFlat
	default:
		return detention.TierRateUnitHour
	}
}
