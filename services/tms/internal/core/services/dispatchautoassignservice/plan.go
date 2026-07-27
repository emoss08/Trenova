package dispatchautoassignservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// PlannedAssignment is one move-to-driver pairing the optimizer chose, carrying the same
// explainable score a dispatcher sees when they pick a driver by hand.
type PlannedAssignment struct {
	MoveID     pulid.ID                                 `json:"moveId"`
	ProNumber  string                                   `json:"proNumber"`
	WorkerID   pulid.ID                                 `json:"workerId"`
	WorkerName string                                   `json:"workerName"`
	TractorID  pulid.ID                                 `json:"tractorId"`
	TrailerID  pulid.ID                                 `json:"trailerId"`
	Score      *dispatchcandidateservice.CandidateScore `json:"score"`
	Confidence decimal.Decimal                          `json:"confidence"`
	Rationale  string                                   `json:"rationale"`
	// AutoExecutable is true only when the organization allows unattended execution and
	// this pairing clears its confidence threshold with no blocking findings.
	AutoExecutable bool     `json:"autoExecutable"`
	ProposalID     pulid.ID `json:"proposalId"`
}

// UncoveredMove is a move the optimizer could not cover, with the reason. Reporting these
// matters more than the successes: a move that silently vanishes from a plan is a load
// nobody is watching.
type UncoveredMove struct {
	MoveID    pulid.ID `json:"moveId"`
	ProNumber string   `json:"proNumber"`
	Reason    string   `json:"reason"`
	// BestBlockedFindings explains the closest candidate's disqualification, so a
	// dispatcher can act on the underlying problem rather than guessing.
	BestBlockedFindings []dispatcheligibility.Finding `json:"bestBlockedFindings"`
}

// Plan is a complete proposed coverage of the board.
type Plan struct {
	RunID       pulid.ID             `json:"runId"`
	Assignments []*PlannedAssignment `json:"assignments"`
	Uncovered   []*UncoveredMove     `json:"uncovered"`
	// ShadowMode reports that nothing was or will be written: the organization is watching
	// what the agent would do before letting it act.
	ShadowMode   bool               `json:"shadowMode"`
	AutonomyTier agent.AutonomyTier `json:"autonomyTier"`
	TotalScore   int                `json:"totalScore"`
	GeneratedAt  int64              `json:"generatedAt"`
}

type PlanRequest struct {
	TenantInfo   pagination.TenantInfo
	WindowStart  int64
	WindowEnd    int64
	FleetCodeIDs []pulid.ID
	MoveIDs      []pulid.ID
	// Apply commits the plan immediately where the organization's autonomy tier allows it.
	// When false the plan is only proposed, whatever the tier.
	Apply bool
}

// costFor converts a candidate score into a minimization cost. Blocked pairings become
// Forbidden so the solver treats them as impossible rather than merely expensive, and an
// over-limit deadhead is treated the same way: an organization that caps empty miles means
// it as a rule, not a preference.
func costFor(
	score *dispatchcandidateservice.CandidateScore,
	control *dispatchcontrol.DispatchControl,
) float64 {
	if score == nil || score.Blocked() {
		return assignmentsolver.Forbidden
	}

	if control != nil && control.AutoAssignMaxDeadheadMiles != nil &&
		score.DeadheadMiles != nil &&
		*score.DeadheadMiles > float64(*control.AutoAssignMaxDeadheadMiles) {
		return assignmentsolver.Forbidden
	}

	// Scores run 0-100 where higher is better; the solver minimizes.
	return float64(100 - score.Score)
}

// confidenceFor expresses how sure the optimizer is about a pairing. It is the score
// itself, discounted for any advisory findings — a driver who is technically eligible but
// carries three warnings should not clear an auto-execute threshold on score alone.
func confidenceFor(score *dispatchcandidateservice.CandidateScore) decimal.Decimal {
	if score == nil || score.Blocked() {
		return decimal.Zero
	}

	base := decimal.NewFromInt(int64(score.Score)).Div(decimal.NewFromInt(100))

	warnings := score.Warnings()
	if warnings == 0 {
		return base
	}

	penalty := decimal.NewFromInt(int64(warnings)).Mul(decimal.NewFromFloat(0.1))
	adjusted := base.Sub(penalty)
	if adjusted.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return adjusted
}

// rationaleFor is the sentence a dispatcher reads when reviewing the proposal. It leads
// with the factors that actually drove the choice rather than restating the score.
func rationaleFor(
	score *dispatchcandidateservice.CandidateScore,
	move *repositories.BoardMove,
) string {
	if score == nil {
		return ""
	}

	rationale := fmt.Sprintf(
		"%s scores %d of 100 for move %s",
		score.WorkerName,
		score.Score,
		move.ProNumber,
	)

	for i, factor := range score.Factors {
		if i >= 3 {
			break
		}
		rationale += fmt.Sprintf("; %s", factor.Detail)
	}

	if warnings := score.Warnings(); warnings > 0 {
		rationale += fmt.Sprintf(" (%d advisory finding(s) to review)", warnings)
	}

	return rationale
}

// evidenceFor records the facts the proposal rests on, so a decision can be audited later
// against what was actually known at the time rather than against current state.
func evidenceFor(
	score *dispatchcandidateservice.CandidateScore,
	move *repositories.BoardMove,
) []agent.EvidenceRef {
	evidence := make([]agent.EvidenceRef, 0, 3)

	evidence = append(evidence, agent.EvidenceRef{
		Type: evidenceTypeMove,
		ID:   move.MoveID.String(),
		Note: fmt.Sprintf(
			"%s → %s, pickup window opens %d",
			originLabel(move),
			destinationLabel(move),
			move.OriginWindowStart,
		),
	})

	evidence = append(evidence, agent.EvidenceRef{
		Type: evidenceTypeHOS,
		ID:   score.WorkerID.String(),
		Note: fmt.Sprintf(
			"drive %dm, shift %dm, cycle %dm remaining",
			score.DriveRemainingMs/60000,
			score.ShiftRemainingMs/60000,
			score.CycleRemainingMs/60000,
		),
	})

	if score.DeadheadMiles != nil {
		evidence = append(evidence, agent.EvidenceRef{
			Type: evidenceTypeDeadhead,
			ID:   score.TractorID.String(),
			Note: fmt.Sprintf(
				"%.0f empty miles to the pickup; %d minutes of appointment margin",
				*score.DeadheadMiles,
				score.MinutesOfSlack,
			),
		})
	}

	return evidence
}

func originLabel(move *repositories.BoardMove) string {
	if move.OriginCity == "" {
		return move.OriginName
	}
	return fmt.Sprintf("%s, %s", move.OriginCity, move.OriginState)
}

func destinationLabel(move *repositories.BoardMove) string {
	if move.DestinationCity == "" {
		return move.DestinationName
	}
	return fmt.Sprintf("%s, %s", move.DestinationCity, move.DestinationState)
}
