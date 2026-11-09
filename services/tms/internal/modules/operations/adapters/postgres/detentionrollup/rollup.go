// Package detentionrollup declares the rollups detention intelligence reads: settled detention
// added up by organization-local day, each grouped only as finely as its readers need, so a read
// costs the days in its window rather than the stops.
package detentionrollup

import (
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
)

const (
	DailyName           = "detention_daily"
	FacilityDailyName   = "detention_facility_daily"
	FacilityMonthlyName = "detention_facility_monthly"
	CustomerDailyName   = "detention_customer_daily"
	CustomerMonthlyName = "detention_customer_monthly"
	PolicyDailyName     = "detention_policy_daily"
	PolicyMonthlyName   = "detention_policy_monthly"
	ArrivalDailyName    = "detention_arrival_daily"
	HoursPerDay         = 24

	Bucket   = "bucket"
	Location = "location_id"
	Customer = "customer_id"
	Policy   = "detention_policy_id"

	Stops           = "row_count"
	DetainedStops   = "detained_stops"
	PastFree        = "past_free_minutes"
	DockMinutes     = "dock_minutes"
	OpenMinutes     = "open_minutes"
	EarlyMinutes    = "early_minutes"
	LateMinutes     = "late_minutes"
	DwellTotal      = "dwell_minutes"
	Billable        = "billable_amount"
	DriverPay       = "driver_pay_amount"
	Net             = "net_margin"
	Waived          = "waived_amount"
	Disputes        = "disputed_stops"
	Suppressed      = "suppressed_stops"
	NoticesDue      = "notices_due"
	NoticesKept     = "notices_on_time"
	FirstClockStart = "clock_start_at_min"
)

var (
	// DwellBounds are the lower edges of the dwell histogram's bins, in minutes: a quarter hour
	// to four hours, half an hour to twelve, an hour to a day, then a day and a half, two days
	// and three, open past that. A quantile read from them is within half a bin of the true one.
	DwellBounds = dwellBounds()
	// FreeBounds are the lower edges of the free-time histogram's bins, in minutes.
	FreeBounds = []float64{0, 30, 60, 90, 120, 150, 180, 240, 300, 360, 480, 720, 1440}

	RecoveryBuckets = []string{
		detentionports.RecoveryLate,
		detentionports.RecoveryNotice,
		detentionports.RecoveryWaived,
		detentionports.RecoveryDisputed,
		detentionports.RecoveryPending,
		detentionports.RecoveryBilled,
		detentionports.RecoveryOther,
	}

	Splits = []string{
		SplitPickup, SplitDelivery, SplitAppointment, SplitOpen, SplitLive, SplitDrop,
	}
)

const (
	SplitPickup      = "pickup"
	SplitDelivery    = "delivery"
	SplitAppointment = "appointment"
	SplitOpen        = "open"
	SplitLive        = "live"
	SplitDrop        = "drop"
)

// Daily adds up an organization's settled detention by local day: the totals, where the money
// went, the splits and the weeks of a read that names no facility.
var Daily = rollup.MustDefine(DailyName, detentiondb.DetentionOccurrencesSchema(),
	rollup.Day(detentiondb.DetentionOccurrences.ClockStartAt),
	nil,
	rollup.Measures(append(append(figures(), recoveryMeasures()...), splitMeasures()...)...),
	settledOptions()...,
)

// Facility, customer and policy figures are kept by local day and by local month. A read takes the
// window's whole months from the monthly rollup and only the days at its edges from the daily one,
// so a long window costs a row per facility per month, not per day. Each is indexed by its
// facility, so a read narrowed to one facility reaches its rows directly.

// FacilityDaily adds up each facility's local day: the days at a window's edges, and every figure
// of a read narrowed to one facility. Its histograms give each facility's dwell and free-time
// quantiles.
var FacilityDaily = facilityRollup(FacilityDailyName, rollup.Day(clockStart()))

// FacilityMonthly adds up each facility's local month: the whole months of a window.
var FacilityMonthly = facilityRollup(FacilityMonthlyName, rollup.Month(clockStart()))

// CustomerDaily adds up each customer's local day, for the customers whose terms cost the most.
var CustomerDaily = customerRollup(CustomerDailyName, rollup.Day(clockStart()))

// CustomerMonthly adds up each customer's local month.
var CustomerMonthly = customerRollup(CustomerMonthlyName, rollup.Month(clockStart()))

// PolicyDaily counts each facility's stops under each policy by local day and keeps the first
// clock start, so a change of policy partway through a window can be placed on its chart.
var PolicyDaily = policyRollup(PolicyDailyName, rollup.Day(clockStart()))

// PolicyMonthly is PolicyDaily by local month.
var PolicyMonthly = policyRollup(PolicyMonthlyName, rollup.Month(clockStart()))

// ArrivalDaily counts an organization's settled stops by the local day and hour they arrived. It
// is bucketed by arrival, not by clock start, so a truck that arrived the evening before its
// appointment counts on the day and hour it came in. A read narrowed to one facility counts that
// facility's arrivals from its stops.
var ArrivalDaily = rollup.MustDefine(ArrivalDailyName, detentiondb.DetentionOccurrencesSchema(),
	rollup.Day(detentiondb.DetentionOccurrences.ArrivedAt),
	nil,
	arrivalMeasures(),
	rollup.TenantZone(OrganizationZone()),
	rollup.Where(keel.And(
		detentiondb.DetentionOccurrences.IsOpen.IsFalse(),
		detentiondb.DetentionOccurrences.ArrivedAt.IsNotNull(),
	)),
	rollup.WithoutSourceIndex(),
)

// All is every detention rollup.
func All() []*rollup.Rollup {
	return []*rollup.Rollup{
		Daily,
		FacilityDaily, FacilityMonthly,
		CustomerDaily, CustomerMonthly,
		PolicyDaily, PolicyMonthly,
		ArrivalDaily,
	}
}

func clockStart() keel.Expr { return detentiondb.DetentionOccurrences.ClockStartAt }

func facilityRollup(name string, grain rollup.Grain) *rollup.Rollup {
	location := detentiondb.DetentionOccurrences.LocationID
	return rollup.MustDefine(name, detentiondb.DetentionOccurrencesSchema(), grain,
		rollup.Dims(location),
		facilityMeasures(),
		append(settledOptions(), rollup.IndexBy(location))...,
	)
}

func customerRollup(name string, grain rollup.Grain) *rollup.Rollup {
	return rollup.MustDefine(name, detentiondb.DetentionOccurrencesSchema(), grain,
		rollup.Dims(detentiondb.DetentionOccurrences.CustomerID),
		figures(),
		settledOptions()...,
	)
}

func policyRollup(name string, grain rollup.Grain) *rollup.Rollup {
	o := detentiondb.DetentionOccurrences
	return rollup.MustDefine(name, detentiondb.DetentionOccurrencesSchema(), grain,
		rollup.Dims(o.LocationID, o.DetentionPolicyID),
		rollup.Measures(rollup.Count(), rollup.Min(o.ClockStartAt)),
		rollup.TenantZone(OrganizationZone()),
		rollup.Where(keel.And(o.IsOpen.IsFalse(), o.DetentionPolicyID.IsNotNull())),
		rollup.IndexBy(o.LocationID),
		rollup.WithoutSourceIndex(),
	)
}

func settledOptions() []rollup.Option {
	return []rollup.Option{
		rollup.TenantZone(OrganizationZone()),
		rollup.Where(detentiondb.DetentionOccurrences.IsOpen.IsFalse()),
		rollup.WithoutSourceIndex(),
	}
}

func dwellBounds() []float64 {
	var out []float64
	for m := 0; m <= 240; m += 15 {
		out = append(out, float64(m))
	}
	for m := 270; m <= 720; m += 30 {
		out = append(out, float64(m))
	}
	for m := 780; m <= 1440; m += 60 {
		out = append(out, float64(m))
	}

	return append(out, 2160, 2880, 4320)
}

// OrganizationZone is the time zone of the row's organization, which its days are counted in.
func OrganizationZone() keel.StringExpr {
	return rollupread.OrganizationZone(detentiondb.DetentionOccurrences.OrganizationID)
}

func DwellBin(i int) string { return fmt.Sprintf("dwell_bin_%d", i) }

func FreeBin(i int) string { return fmt.Sprintf("free_bin_%d", i) }

func ArrivalsAt(hour int) string { return fmt.Sprintf("arrivals_h%02d", hour) }

func DetainedArrivalsAt(hour int) string { return fmt.Sprintf("detained_arrivals_h%02d", hour) }

func RecoveryCount(bucket string) string { return "recovery_" + strings.ToLower(bucket) + "_stops" }

func RecoveryGross(bucket string) string { return "recovery_" + strings.ToLower(bucket) + "_gross" }

func RecoveryBillable(bucket string) string {
	return "recovery_" + strings.ToLower(bucket) + "_billable"
}

func SplitStops(split string) string { return "split_" + split + "_stops" }

func SplitDetained(split string) string { return "split_" + split + "_detained" }

func SplitMinutes(split string) string { return "split_" + split + "_minutes" }

// figures are what every figure of a day adds up to: stops, the wait past free time by cause,
// the money and the notices.
func figures() []rollup.Measure {
	o := detentiondb.DetentionOccurrences
	late, open, early, dock := Causes()
	pastFree := PastFreeMinutes()

	return rollup.Measures(
		rollup.Count(),
		rollup.CountWhere(DetainedStops, Detained()),
		rollup.SumOf(PastFree, pastFree),
		rollup.SumWhere(DockMinutes, pastFree, dock),
		rollup.SumWhere(OpenMinutes, pastFree, open),
		rollup.SumWhere(EarlyMinutes, pastFree, early),
		rollup.SumWhere(LateMinutes, pastFree, late),
		rollup.SumOf(DwellTotal, o.RawDwellMinutes),
		rollup.SumOf(Billable, o.BillableAmount),
		rollup.SumOf(DriverPay, o.DriverPayAmount),
		rollup.SumOf(Net, o.NetMargin),
		rollup.SumOf(Waived, o.WaivedAmount),
		rollup.CountWhere(Disputes, o.Status.Eq(detention.OccurrenceStatusDisputed)),
		rollup.CountWhere(Suppressed, o.SuppressedByGate.IsTrue()),
		rollup.CountWhere(NoticesDue, NoticeDue()),
		rollup.CountWhere(NoticesKept, NoticeOnTime()),
	)
}

func recoveryMeasures() []rollup.Measure {
	o := detentiondb.DetentionOccurrences
	bucket := RecoveryBucket()
	out := make([]rollup.Measure, 0, 3*len(RecoveryBuckets))
	for _, b := range RecoveryBuckets {
		in := bucket.Eq(b)
		out = append(out,
			rollup.CountWhere(RecoveryCount(b), in),
			rollup.SumWhere(RecoveryGross(b), o.GrossAmount, in),
			rollup.SumWhere(RecoveryBillable(b), o.BillableAmount, in),
		)
	}

	return out
}

func splitMeasures() []rollup.Measure {
	pastFree := PastFreeMinutes()
	out := make([]rollup.Measure, 0, 3*len(Splits))
	for _, split := range Splits {
		in := splitPred(split)
		out = append(out,
			rollup.CountWhere(SplitStops(split), in),
			rollup.CountWhere(SplitDetained(split), keel.And(in, Detained())),
			rollup.SumWhere(SplitMinutes(split), pastFree, in),
		)
	}

	return out
}

func facilityMeasures() []rollup.Measure {
	o := detentiondb.DetentionOccurrences
	late, _, _, _ := Causes()
	out := append(append(figures(), recoveryMeasures()...), splitMeasures()...)
	out = append(out, rollup.Histogram("dwell_bin", o.RawDwellMinutes, DwellBounds, nil)...)
	// A late arrival forfeits its free time and records none, which would pull the median to
	// zero, so free time is read from the stops that kept it.
	out = append(out,
		rollup.Histogram("free_bin", o.FreeMinutesGranted, FreeBounds, keel.Not(late))...)

	return out
}

func arrivalMeasures() []rollup.Measure {
	o := detentiondb.DetentionOccurrences
	hour := rollupread.LocalHour(o.ArrivedAt, OrganizationZone())
	out := make([]rollup.Measure, 0, 2*HoursPerDay)
	for h := range HoursPerDay {
		at := hour.Eq(int64(h))
		out = append(out,
			rollup.CountWhere(ArrivalsAt(h), at),
			rollup.CountWhere(DetainedArrivalsAt(h), keel.And(at, Detained())),
		)
	}

	return out
}

func splitPred(split string) keel.Pred {
	o := detentiondb.DetentionOccurrences
	switch split {
	case SplitPickup:
		return o.StopType.In(shipmentvocab.StopTypePickup, shipmentvocab.StopTypeSplitPickup)
	case SplitDelivery:
		return o.StopType.In(shipmentvocab.StopTypeDelivery, shipmentvocab.StopTypeSplitDelivery)
	case SplitAppointment:
		return o.ScheduleType.Eq(shipmentvocab.StopScheduleTypeAppointment)
	case SplitOpen:
		return o.ScheduleType.Eq(shipmentvocab.StopScheduleTypeOpen)
	case SplitLive:
		return o.Handling.Eq(shipmentvocab.StopHandlingLive)
	default:
		return o.Handling.Eq(shipmentvocab.StopHandlingDrop)
	}
}
