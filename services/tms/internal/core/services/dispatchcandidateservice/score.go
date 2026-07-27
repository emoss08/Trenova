package dispatchcandidateservice

import (
	"fmt"
	"math"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"github.com/emoss08/trenova/shared/pulid"
)

// ScoreFactor is one weighted dimension of a candidate's score, carrying both its
// contribution and the sentence a dispatcher reads to understand it. The console renders
// every factor: a ranking nobody can audit is a ranking nobody uses.
type ScoreFactor struct {
	Key          dispatchcontrol.ScoringFactor `json:"key"`
	Label        string                        `json:"label"`
	Raw          float64                       `json:"raw"`
	Weight       float64                       `json:"weight"`
	Contribution float64                       `json:"contribution"`
	Detail       string                        `json:"detail"`
}

// CandidateScore is one driver-and-equipment pairing judged against one move.
type CandidateScore struct {
	WorkerID   pulid.ID  `json:"workerId"`
	WorkerName string    `json:"workerName"`
	TractorID  pulid.ID  `json:"tractorId"`
	TrailerID  pulid.ID  `json:"trailerId"`
	MoveID     pulid.ID  `json:"moveId"`

	Score   int    `json:"score"`
	Verdict string `json:"verdict"`

	DeadheadMiles     *float64 `json:"deadheadMiles"`
	EstimatedDriveMs  int64    `json:"estimatedDriveMs"`
	ProjectedArrival  int64    `json:"projectedArrival"`
	MinutesOfSlack    int64    `json:"minutesOfSlack"`
	DriveRemainingMs  int64    `json:"driveRemainingMs"`
	ShiftRemainingMs  int64    `json:"shiftRemainingMs"`
	CycleRemainingMs  int64    `json:"cycleRemainingMs"`
	ProjectedAvailable int64   `json:"projectedTimeAvailable"`

	Findings []dispatcheligibility.Finding `json:"findings"`
	Factors  []ScoreFactor                 `json:"factors"`
}

// Blocked reports whether any hard constraint disqualifies this pairing. A blocked
// candidate is still returned, with its reasons, so the console can explain an absence
// instead of silently hiding a driver a dispatcher expected to see.
func (c *CandidateScore) Blocked() bool {
	for i := range c.Findings {
		if c.Findings[i].Severity == dispatcheligibility.SeverityBlock {
			return true
		}
	}
	return false
}

func (c *CandidateScore) Warnings() int {
	count := 0
	for i := range c.Findings {
		if c.Findings[i].Severity == dispatcheligibility.SeverityWarn {
			count++
		}
	}
	return count
}

const (
	// maxScore keeps the weighted sum on a 0-100 scale a dispatcher can read at a glance.
	maxScore = 100.0

	// Normalization anchors. Each soft factor is reduced to a 0-1 goodness value before
	// weighting, so factors measured in miles, hours, and counts stay comparable.
	deadheadFullCreditMiles = 25.0
	deadheadZeroCreditMiles = 250.0
	slackFullCreditMinutes  = 240.0
	hosMarginFullCreditMs   = float64(6 * 3600 * 1000)
	loadBalanceFullMiles    = 3000.0
	laneExperienceFullCount = 10.0
	homeTimeFullCreditDays  = 14.0
)

type factorInput struct {
	deadheadMiles    *float64
	slackMinutes     float64
	hosMarginMs      float64
	trailerContinues bool
	trailerKnown     bool
	fleetMatches     bool
	fleetKnown       bool
	driverTypeFit    float64
	driverTypeLabel  string
	daysOut          float64
	daysOutKnown     bool
	committedMiles   float64
	laneMoves        float64
	customerName     string
}

// buildFactors reduces every soft signal to a 0-1 goodness value, applies the
// organization's weights, and returns both the total and the per-factor breakdown.
// Factors with no data are omitted rather than scored as zero, so a missing telematics
// feed does not quietly push a good driver down the list.
func buildFactors(in factorInput, weights map[dispatchcontrol.ScoringFactor]float64) (int, []ScoreFactor) {
	factors := make([]ScoreFactor, 0, len(dispatchcontrol.AllScoringFactors()))

	add := func(key dispatchcontrol.ScoringFactor, label string, raw float64, detail string) {
		weight := weights[key]
		if weight <= 0 {
			return
		}
		factors = append(factors, ScoreFactor{
			Key:    key,
			Label:  label,
			Raw:    clamp01(raw),
			Weight: weight,
			Detail: detail,
		})
	}

	if in.deadheadMiles != nil {
		add(
			dispatchcontrol.FactorDeadhead,
			"Empty miles",
			invertedRamp(*in.deadheadMiles, deadheadFullCreditMiles, deadheadZeroCreditMiles),
			fmt.Sprintf("%.0f empty miles to the pickup", *in.deadheadMiles),
		)
	}

	add(
		dispatchcontrol.FactorHOSMargin,
		"Hours remaining",
		clamp01(in.hosMarginMs/hosMarginFullCreditMs),
		fmt.Sprintf("%s of clock left after the trip", formatDurationMs(int64(in.hosMarginMs))),
	)

	add(
		dispatchcontrol.FactorOnTime,
		"On-time margin",
		clamp01(in.slackMinutes/slackFullCreditMinutes),
		describeSlack(in.slackMinutes),
	)

	if in.trailerKnown {
		add(
			dispatchcontrol.FactorTrailerContinuity,
			"Trailer continuity",
			boolScore(in.trailerContinues),
			trailerContinuityDetail(in.trailerContinues),
		)
	}

	if in.fleetKnown {
		add(
			dispatchcontrol.FactorFleetMatch,
			"Fleet match",
			boolScore(in.fleetMatches),
			fleetMatchDetail(in.fleetMatches),
		)
	}

	if in.driverTypeLabel != "" {
		add(
			dispatchcontrol.FactorDriverTypeFit,
			"Haul fit",
			in.driverTypeFit,
			in.driverTypeLabel,
		)
	}

	if in.daysOutKnown {
		add(
			dispatchcontrol.FactorHomeTime,
			"Home time",
			invertedRamp(in.daysOut, 0, homeTimeFullCreditDays),
			fmt.Sprintf("%.0f days since last home", in.daysOut),
		)
	}

	add(
		dispatchcontrol.FactorLoadBalance,
		"Load balance",
		invertedRamp(in.committedMiles, 0, loadBalanceFullMiles),
		fmt.Sprintf("%.0f miles committed this period", in.committedMiles),
	)

	if in.laneMoves > 0 {
		add(
			dispatchcontrol.FactorLaneExperience,
			"Customer experience",
			clamp01(in.laneMoves/laneExperienceFullCount),
			describeLaneExperience(in.laneMoves, in.customerName),
		)
	}

	return finalizeFactors(factors)
}

// finalizeFactors normalizes the applied weights so the score always spans 0-100 no
// matter how many factors had data, then records each factor's share of the result.
func finalizeFactors(factors []ScoreFactor) (int, []ScoreFactor) {
	if len(factors) == 0 {
		return 0, factors
	}

	totalWeight := 0.0
	for i := range factors {
		totalWeight += factors[i].Weight
	}
	if totalWeight <= 0 {
		return 0, factors
	}

	total := 0.0
	for i := range factors {
		contribution := factors[i].Raw * factors[i].Weight / totalWeight * maxScore
		factors[i].Contribution = round1(contribution)
		total += contribution
	}

	sort.SliceStable(factors, func(i, j int) bool {
		return factors[i].Contribution > factors[j].Contribution
	})

	return int(math.Round(clampScore(total))), factors
}

// driverTypeFit rewards matching the length of haul to how the driver runs. A local
// driver on a 900-mile lane is technically legal and operationally wrong.
func driverTypeFit(driverType worker.DriverType, tripMiles float64) (float64, string) {
	if tripMiles <= 0 {
		return 0.5, ""
	}

	switch driverType {
	case worker.DriverTypeLocal:
		return rampFit(tripMiles, 150, 300), fmt.Sprintf("Local driver on a %.0f mile run", tripMiles)
	case worker.DriverTypeRegional:
		return rampFit(tripMiles, 500, 800), fmt.Sprintf("Regional driver on a %.0f mile run", tripMiles)
	case worker.DriverTypeOTR, worker.DriverTypeTeam:
		if tripMiles < 100 {
			return 0.4, fmt.Sprintf("Over-the-road driver on a short %.0f mile run", tripMiles)
		}
		return 1, fmt.Sprintf("Over-the-road driver on a %.0f mile run", tripMiles)
	default:
		return 0.5, ""
	}
}

// rampFit scores 1 below the comfortable ceiling and decays to 0 at the hard ceiling.
func rampFit(value, comfortable, hard float64) float64 {
	if value <= comfortable {
		return 1
	}
	if value >= hard {
		return 0
	}
	return 1 - (value-comfortable)/(hard-comfortable)
}

// invertedRamp scores 1 at or below best and 0 at or above worst — for metrics where
// smaller is better, such as empty miles.
func invertedRamp(value, best, worst float64) float64 {
	if worst <= best {
		return 0
	}
	if value <= best {
		return 1
	}
	if value >= worst {
		return 0
	}
	return 1 - (value-best)/(worst-best)
}

func boolScore(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampScore(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > maxScore {
		return maxScore
	}
	return v
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func describeSlack(minutes float64) string {
	switch {
	case minutes < 0:
		return fmt.Sprintf("Projected %.0f minutes late to the pickup", -minutes)
	case minutes < 60:
		return fmt.Sprintf("Only %.0f minutes of margin to the appointment", minutes)
	default:
		return fmt.Sprintf("%.1f hours of margin to the appointment", minutes/60)
	}
}

func trailerContinuityDetail(continues bool) string {
	if continues {
		return "Keeps the trailer from the previous leg"
	}
	return "Requires a trailer swap"
}

func fleetMatchDetail(matches bool) string {
	if matches {
		return "Driver and power unit are in the same fleet"
	}
	return "Driver and power unit are in different fleets"
}

func describeLaneExperience(moves float64, customerName string) string {
	if customerName == "" {
		return fmt.Sprintf("Has run this customer %.0f times", moves)
	}
	return fmt.Sprintf("Has run %s %.0f times", customerName, moves)
}

func formatDurationMs(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	hours := ms / 3_600_000
	minutes := (ms % 3_600_000) / 60_000
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

// verdictFor maps the eligibility findings and the remaining margin onto the same
// vocabulary the existing shipment-level feasibility check already uses, so the console
// and the assignment dialog never disagree about what "tight" means.
func verdictFor(eval *dispatcheligibility.Evaluation, slackMinutes float64, hosKnown bool) string {
	if eval.Blocked() {
		return telematicsservice.FeasibilityVerdictInfeasible
	}
	if !hosKnown {
		return telematicsservice.FeasibilityVerdictUnknown
	}
	if slackMinutes < 0 {
		return telematicsservice.FeasibilityVerdictInfeasible
	}
	if slackMinutes < 90 {
		return telematicsservice.FeasibilityVerdictTight
	}
	return telematicsservice.FeasibilityVerdictFeasible
}
