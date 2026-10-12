package worker

import "github.com/emoss08/trenova/shared/pulid"

// FleetSafetyBasic is one BASIC's standing across the fleet. The weighted
// score is the FMCSA's own arithmetic — severity, plus two for an out-of-
// service order, times a recency multiplier — so a number here means what it
// means on a Safety Measurement System report.
type FleetSafetyBasic struct {
	Basic         CSABasic
	Violations    int32
	Events        int32
	WeightedScore int32
	OutOfService  int32
	// Inferred says the BASIC was reached from the kind of event rather than
	// from violations somebody keyed in. A safety director reading a number
	// should know whether it came off an inspection report or off a guess.
	Inferred bool
}

// FleetSafetyKind counts one kind of event across the window.
type FleetSafetyKind struct {
	Kind         SafetyEventKind
	Events       int32
	Points       int32
	Preventable  int32
	OutOfService int32
	Open         int32
}

// FleetSafetyTerminal is one terminal's standing, read from the roster cache
// so the fleet view and the roster list cannot disagree.
type FleetSafetyTerminal struct {
	FleetCodeID  pulid.ID
	Code         string
	Description  string
	Color        string
	Workers      int32
	AtRisk       int32
	Watch        int32
	AverageScore int32
}

// FleetSafetyTrendPoint is one month of the trend line.
type FleetSafetyTrendPoint struct {
	PeriodStart  int64
	Events       int32
	Accidents    int32
	Preventable  int32
	Citations    int32
	Inspections  int32
	OutOfService int32
	Points       int32
}

// FleetSafetyRank is one driver's line in the ranking.
type FleetSafetyRank struct {
	WorkerID     pulid.ID
	Name         string
	FleetCodeID  pulid.ID
	FleetCode    string
	FleetColor   string
	Rating       SafetyRating
	Score        int32
	ActivePoints int32
	Events       int32
	LastEventAt  *int64
}

// FleetSafetyRatingCount is how many of the roster sit in one rating.
type FleetSafetyRatingCount struct {
	Rating  SafetyRating
	Workers int32
}

// FleetSafety is the whole fleet's safety picture: who is where, what has been
// happening, and which BASICs are carrying the weight.
type FleetSafety struct {
	AsOf         int64
	WindowMonths int32
	Workers      int32
	AverageScore int32
	AtRisk       int32
	Watch        int32
	Ratings      []FleetSafetyRatingCount
	Basics       []FleetSafetyBasic
	Kinds        []FleetSafetyKind
	Terminals    []FleetSafetyTerminal
	Trend        []FleetSafetyTrendPoint
	Worst        []FleetSafetyRank
	Best         []FleetSafetyRank
	// TotalEvents and TotalPoints are the window's totals, so the header does
	// not have to be summed from the kind breakdown by every caller.
	TotalEvents        int32
	TotalPoints        int32
	OpenEvents         int32
	OutOfServiceOrders int32
	// BasicsInferred is true when any BASIC figure was reached from the kind
	// of event rather than from recorded violations. It is surfaced rather
	// than hidden: a scorecard nobody keyed violations into is an estimate,
	// and saying so is the difference between a useful number and a false one.
	BasicsInferred bool
}

// csaBucketWeights maps the recency bucket the query grouped by onto the
// FMCSA's multipliers. The buckets come back from SQL and the weights live
// here, so the arithmetic can be tested without a database.
var csaBucketWeights = [3]int32{csaWeightRecent, csaWeightMid, csaWeightOld}

// CSABucketWeight is the multiplier for a recency bucket: 0 is inside six
// months, 1 inside a year, 2 out to the two-year look-back.
func CSABucketWeight(bucket int) int32 {
	if bucket < 0 || bucket >= len(csaBucketWeights) {
		return 0
	}
	return csaBucketWeights[bucket]
}

// FleetSafetyBasicInput is one BASIC's violations in one recency bucket.
type FleetSafetyBasicInput struct {
	Basic        CSABasic
	Bucket       int
	Violations   int32
	SeveritySum  int32
	OutOfService int32
}

// FleetSafetyEventBasicInput is one bucket of events nobody keyed violations
// into, grouped by the kind and result the BASIC is inferred from.
type FleetSafetyEventBasicInput struct {
	Kind             SafetyEventKind
	InspectionResult InspectionResult
	Bucket           int
	Events           int32
	Points           int32
	OutOfService     int32
}

// BuildFleetBasics folds recorded violations and inferred events into one
// figure per BASIC, in the FMCSA's own display order.
//
// Both sources are needed. A carrier who keys violation codes gets the real
// thing; one who only records that an inspection failed still gets a
// scorecard, marked as inferred so nobody mistakes it for one.
func BuildFleetBasics(
	violations []FleetSafetyBasicInput,
	events []FleetSafetyEventBasicInput,
) []FleetSafetyBasic {
	byBasic := make(map[CSABasic]*FleetSafetyBasic, len(AllCSABasics()))
	for _, basic := range AllCSABasics() {
		byBasic[basic] = &FleetSafetyBasic{Basic: basic}
	}

	for _, row := range violations {
		entry, ok := byBasic[row.Basic]
		if !ok {
			continue
		}
		weight := CSABucketWeight(row.Bucket)
		entry.Violations += row.Violations
		entry.OutOfService += row.OutOfService
		// Out of service adds two to each violation carrying it, which is why
		// the flag is counted separately from the severity sum.
		entry.WeightedScore += (row.SeveritySum + row.OutOfService*csaOutOfServiceAdd) * weight
	}

	for _, row := range events {
		basic := SuggestedBasic(row.Kind, row.InspectionResult)
		entry, ok := byBasic[basic]
		if !ok || basic == "" {
			continue
		}
		weight := CSABucketWeight(row.Bucket)
		entry.Events += row.Events
		entry.OutOfService += row.OutOfService
		entry.WeightedScore += (row.Points + row.OutOfService*csaOutOfServiceAdd) * weight
		if row.Events > 0 {
			entry.Inferred = true
		}
	}

	out := make([]FleetSafetyBasic, 0, len(byBasic))
	for _, basic := range AllCSABasics() {
		out = append(out, *byBasic[basic])
	}
	return out
}

// AverageScore is the mean safety score over a headcount, rounded to the
// nearest whole. Zero workers gives zero rather than a divide: a fleet with
// nobody in it does not have a perfect record, it has no record.
func AverageScore(totalScore, workers int32) int32 {
	if workers <= 0 {
		return 0
	}
	return (totalScore + workers/2) / workers
}
