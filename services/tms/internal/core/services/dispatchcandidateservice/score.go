package dispatchcandidateservice

import (
	"fmt"
	"math"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/hosprojection"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ScoreFactor struct {
	Key          dispatchcontrol.ScoringFactor `json:"key"`
	Label        string                        `json:"label"`
	Raw          float64                       `json:"raw"`
	Weight       float64                       `json:"weight"`
	Contribution float64                       `json:"contribution"`
	Detail       string                        `json:"detail"`
}

type CandidateScore struct {
	WorkerID   pulid.ID `json:"workerId"`
	WorkerName string   `json:"workerName"`
	TractorID  pulid.ID `json:"tractorId"`
	TrailerID  pulid.ID `json:"trailerId"`
	MoveID     pulid.ID `json:"moveId"`

	Score   int    `json:"score"`
	Verdict string `json:"verdict"`

	DeadheadMiles      *float64 `json:"deadheadMiles"`
	EstimatedDriveMs   int64    `json:"estimatedDriveMs"`
	ProjectedArrival   int64    `json:"projectedArrival"`
	MinutesOfSlack     int64    `json:"minutesOfSlack"`
	DriveRemainingMs   int64    `json:"driveRemainingMs"`
	ShiftRemainingMs   int64    `json:"shiftRemainingMs"`
	CycleRemainingMs   int64    `json:"cycleRemainingMs"`
	ProjectedAvailable int64    `json:"projectedTimeAvailable"`

	HOSStrategy          string `json:"hosStrategy"`
	HOSRestStartDeadline int64  `json:"hosRestStartDeadline"`
	HOSProjectedDriveMs  int64  `json:"hosProjectedDriveMs"`
	HOSProjectedShiftMs  int64  `json:"hosProjectedShiftMs"`
	HOSProjectedCycleMs  int64  `json:"hosProjectedCycleMs"`

	Findings []dispatcheligibility.Finding `json:"findings"`
	Factors  []ScoreFactor                 `json:"factors"`
}

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
	maxScore = 100.0

	deadheadFullCreditMiles = 25.0
	deadheadZeroCreditMiles = 250.0
	slackFullCreditMinutes  = 240.0
	hosMarginFullCreditMs   = float64(6 * 3600 * 1000)
	loadBalanceFullMiles    = 3000.0
	laneExperienceFullCount = 10.0
	homeTimeZeroCreditDays  = 14.0

	onTimeHistoryMinStops           = 5
	onTimeShortfallZeroCreditPp     = 20.0
	defaultOnTimeTargetPct          = 95.0
	onTimeFailurePenaltyPerIncident = 0.05
	safetyZeroCreditViolations      = 5.0
	acceptanceMinDecisions          = 3
	ackLatencyFullCreditMinutes     = 15.0
	ackLatencyZeroCreditMinutes     = 240.0
	ptoGapFullCreditHours           = 48.0
	loadBalanceFullCreditMoves      = 10.0
	openAssignmentsZeroCredit       = 4.0
)

type factorInput struct {
	deadheadMiles    *float64
	slackMinutes     float64
	slackKnown       bool
	hosMarginMs      float64
	hosKnown         bool
	hosDetail        string
	trailerContinues bool
	trailerKnown     bool
	fleetMatches     bool
	fleetKnown       bool
	driverTypeFit    float64
	driverTypeLabel  string
	daysOut          float64
	daysOutKnown     bool
	committedMiles   float64
	moveCount        float64
	openAssignments  float64
	workloadKnown    bool
	laneMoves        float64
	customerName     string

	onTimeKnown         bool
	onTimePct           float64
	onTimeStops         int
	onTimeTargetPct     float64
	driverFaultFailures int
	safetyKnown         bool
	violationCount      float64
	acceptanceKnown     bool
	acceptRate          float64
	ackDecisions        int
	ackLatencyMinutes   float64
	ptoKnown            bool
	ptoGapHours         float64
}

type factorList struct {
	weights map[dispatchcontrol.ScoringFactor]float64
	factors []ScoreFactor
}

func (l *factorList) add(
	key dispatchcontrol.ScoringFactor,
	label string,
	raw float64,
	detail string,
) {
	weight := l.weights[key]
	if weight <= 0 {
		return
	}
	l.factors = append(l.factors, ScoreFactor{
		Key:    key,
		Label:  label,
		Raw:    clamp01(raw),
		Weight: weight,
		Detail: detail,
	})
}

func buildFactors(
	in *factorInput,
	weights map[dispatchcontrol.ScoringFactor]float64,
) (int, []ScoreFactor) {
	list := &factorList{
		weights: weights,
		factors: make([]ScoreFactor, 0, len(dispatchcontrol.AllScoringFactors())),
	}

	addTripFactors(list, in)
	addDriverContextFactors(list, in)
	addPerformanceFactors(list, in)

	return finalizeFactors(list.factors)
}

func addTripFactors(list *factorList, in *factorInput) {
	if in.deadheadMiles != nil {
		list.add(
			dispatchcontrol.FactorDeadhead,
			"Empty miles",
			invertedRamp(*in.deadheadMiles, deadheadFullCreditMiles, deadheadZeroCreditMiles),
			fmt.Sprintf("%.0f empty miles to the pickup", *in.deadheadMiles),
		)
	}

	if in.hosKnown {
		detail := in.hosDetail
		if detail == "" {
			detail = fmt.Sprintf(
				"%s of clock left after the trip",
				timeutils.FormatDurationMs(int64(in.hosMarginMs)),
			)
		}
		list.add(
			dispatchcontrol.FactorHOSMargin,
			"Hours remaining",
			clamp01(in.hosMarginMs/hosMarginFullCreditMs),
			detail,
		)
	}

	if in.slackKnown {
		list.add(
			dispatchcontrol.FactorOnTime,
			"On-time margin",
			clamp01(in.slackMinutes/slackFullCreditMinutes),
			describeSlack(in.slackMinutes),
		)
	}
}

func addDriverContextFactors(list *factorList, in *factorInput) {
	if in.trailerKnown {
		list.add(
			dispatchcontrol.FactorTrailerContinuity,
			"Trailer continuity",
			boolScore(in.trailerContinues),
			trailerContinuityDetail(in.trailerContinues),
		)
	}

	if in.fleetKnown {
		list.add(
			dispatchcontrol.FactorFleetMatch,
			"Fleet match",
			boolScore(in.fleetMatches),
			fleetMatchDetail(in.fleetMatches),
		)
	}

	if in.driverTypeLabel != "" {
		list.add(
			dispatchcontrol.FactorDriverTypeFit,
			"Haul fit",
			in.driverTypeFit,
			in.driverTypeLabel,
		)
	}

	if in.daysOutKnown {
		list.add(
			dispatchcontrol.FactorHomeTime,
			"Home time",
			invertedRamp(in.daysOut, 0, homeTimeZeroCreditDays),
			fmt.Sprintf("%.0f days since last home", in.daysOut),
		)
	}

	if in.workloadKnown {
		raw := 0.5*invertedRamp(in.committedMiles, 0, loadBalanceFullMiles) +
			0.3*invertedRamp(in.moveCount, 0, loadBalanceFullCreditMoves) +
			0.2*invertedRamp(in.openAssignments, 0, openAssignmentsZeroCredit)
		list.add(
			dispatchcontrol.FactorLoadBalance,
			"Load balance",
			raw,
			fmt.Sprintf(
				"%.0f mi, %.0f moves this week; %.0f open assignment(s)",
				in.committedMiles,
				in.moveCount,
				in.openAssignments,
			),
		)
	}

	if in.laneMoves > 0 {
		list.add(
			dispatchcontrol.FactorLaneExperience,
			"Customer experience",
			clamp01(in.laneMoves/laneExperienceFullCount),
			describeLaneExperience(in.laneMoves, in.customerName),
		)
	}
}

func addPerformanceFactors(list *factorList, in *factorInput) {
	if in.onTimeKnown {
		raw := onTimeRamp(in.onTimePct, in.onTimeTargetPct) -
			onTimeFailurePenaltyPerIncident*float64(in.driverFaultFailures)
		list.add(
			dispatchcontrol.FactorOnTimeHistory,
			"On-time history",
			raw,
			describeOnTimeHistory(in),
		)
	}

	if in.safetyKnown {
		list.add(
			dispatchcontrol.FactorSafetyHistory,
			"Safety history",
			invertedRamp(in.violationCount, 0, safetyZeroCreditViolations),
			describeSafetyHistory(in.violationCount),
		)
	}

	if in.acceptanceKnown {
		latencyRamp := 1.0
		if in.ackLatencyMinutes > 0 {
			latencyRamp = invertedRamp(
				in.ackLatencyMinutes,
				ackLatencyFullCreditMinutes,
				ackLatencyZeroCreditMinutes,
			)
		}
		list.add(
			dispatchcontrol.FactorAcceptance,
			"Acceptance",
			0.8*in.acceptRate+0.2*latencyRamp,
			describeAcceptance(in),
		)
	}

	if in.ptoKnown {
		list.add(
			dispatchcontrol.FactorPTOProximity,
			"Time-off buffer",
			clamp01(in.ptoGapHours/ptoGapFullCreditHours),
			describePTOProximity(in.ptoGapHours),
		)
	}
}

// onTimeRamp gives full credit at or above the organization's on-time goal and no
// credit twenty points below it.
func onTimeRamp(pct, target float64) float64 {
	return invertedRamp(target-pct, 0, onTimeShortfallZeroCreditPp)
}

func describeOnTimeHistory(in *factorInput) string {
	detail := fmt.Sprintf(
		"%.0f%% on-time over %d stops (goal %.0f%%)",
		in.onTimePct,
		in.onTimeStops,
		in.onTimeTargetPct,
	)
	if in.driverFaultFailures > 0 {
		detail += fmt.Sprintf("; %d driver-fault service failure(s)", in.driverFaultFailures)
	}
	return detail
}

func describeSafetyHistory(violations float64) string {
	if violations == 0 {
		return "No HOS violations in the last 90 days"
	}
	return fmt.Sprintf("%.0f HOS violation(s) in the last 90 days", violations)
}

func describeAcceptance(in *factorInput) string {
	accepted := int(math.Round(in.acceptRate * float64(in.ackDecisions)))
	detail := fmt.Sprintf("Accepted %d of %d assignments", accepted, in.ackDecisions)
	if in.ackLatencyMinutes > 0 {
		detail += fmt.Sprintf(", typically within %.0fm", in.ackLatencyMinutes)
	}
	return detail
}

func describePTOProximity(gapHours float64) string {
	if gapHours < 0 {
		return fmt.Sprintf(
			"Run projected to overrun approved time off by %s",
			timeutils.FormatDurationMs(int64(-gapHours*3_600_000)),
		)
	}
	return fmt.Sprintf(
		"Run projected to finish %s before approved time off",
		timeutils.FormatDurationMs(int64(gapHours*3_600_000)),
	)
}

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

func driverTypeFit(driverType worker.DriverType, tripMiles float64) (fit float64, label string) {
	if tripMiles <= 0 {
		return 0.5, ""
	}

	switch driverType {
	case worker.DriverTypeLocal:
		return rampFit(
				tripMiles,
				150,
				300,
			), fmt.Sprintf(
				"Local driver on a %.0f mile run",
				tripMiles,
			)
	case worker.DriverTypeRegional:
		return rampFit(
				tripMiles,
				500,
				800,
			), fmt.Sprintf(
				"Regional driver on a %.0f mile run",
				tripMiles,
			)
	case worker.DriverTypeOTR, worker.DriverTypeTeam:
		if tripMiles < 100 {
			return 0.4, fmt.Sprintf("Over-the-road driver on a short %.0f mile run", tripMiles)
		}
		return 1, fmt.Sprintf("Over-the-road driver on a %.0f mile run", tripMiles)
	default:
		return 0.5, ""
	}
}

func rampFit(value, comfortable, hard float64) float64 {
	if value <= comfortable {
		return 1
	}
	if value >= hard {
		return 0
	}
	return 1 - (value-comfortable)/(hard-comfortable)
}

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
		return fmt.Sprintf(
			"Projected %s late to the pickup",
			timeutils.FormatLongDurationMs(int64(-minutes*60_000)),
		)
	case minutes < 60:
		return fmt.Sprintf("Only %.0f minutes of margin to the appointment", minutes)
	default:
		return fmt.Sprintf(
			"%s of margin to the appointment",
			timeutils.FormatLongDurationMs(int64(minutes*60_000)),
		)
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

type verdictInput struct {
	Eval         *dispatcheligibility.Evaluation
	SlackMinutes float64
	HOSKnown     bool
	HOSExpected  bool
	Projection   *hosprojection.Result
}

// verdictFor folds eligibility, appointment slack, and the HOS projection into one
// feasibility verdict. Unknown is reserved for organizations that do track hours but
// have no fresh data for this driver — an organization without telematics is judged
// on schedule alone, never penalized for data it cannot have. A trip that is legal
// only after additional rest is at best tight.
func verdictFor(in *verdictInput) string {
	if in.Eval.Blocked() {
		return telematics.FeasibilityVerdictInfeasible
	}
	if in.HOSExpected && !in.HOSKnown {
		return telematics.FeasibilityVerdictUnknown
	}
	if in.Projection != nil && !in.Projection.Feasible {
		return telematics.FeasibilityVerdictInfeasible
	}
	if in.SlackMinutes < 0 {
		return telematics.FeasibilityVerdictInfeasible
	}
	if in.SlackMinutes < 90 {
		return telematics.FeasibilityVerdictTight
	}
	if in.Projection != nil &&
		in.Projection.Strategy != hosprojection.StrategyCurrentClocks {
		return telematics.FeasibilityVerdictTight
	}
	return telematics.FeasibilityVerdictFeasible
}
