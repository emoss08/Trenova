package dispatchautoassignservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/shopspring/decimal"
)

func costFor(
	score *dispatchcandidateservice.CandidateScore,
	control *dispatchcontrol.DispatchControl,
) float64 {
	if score == nil || score.Blocked() {
		return assignmentsolver.Forbidden
	}

	if deadheadExceedsLimit(score, control) {
		return assignmentsolver.Forbidden
	}

	return float64(100 - score.Score)
}

func deadheadExceedsLimit(
	score *dispatchcandidateservice.CandidateScore,
	control *dispatchcontrol.DispatchControl,
) bool {
	return control != nil && control.AutoAssignMaxDeadheadMiles != nil &&
		score.DeadheadMiles != nil &&
		*score.DeadheadMiles > float64(*control.AutoAssignMaxDeadheadMiles)
}

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

func rationaleFor(
	score *dispatchcandidateservice.CandidateScore,
	move *repositories.BoardMove,
) string {
	if score == nil {
		return ""
	}

	var rationale strings.Builder
	fmt.Fprintf(
		&rationale,
		"%s scores %d of 100 for move %s",
		score.WorkerName,
		score.Score,
		move.ProNumber,
	)

	for i, factor := range score.Factors {
		if i >= 3 {
			break
		}
		rationale.WriteString("; ")
		rationale.WriteString(factor.Detail)
	}

	if warnings := score.Warnings(); warnings > 0 {
		fmt.Fprintf(&rationale, " (%d advisory finding(s) to review)", warnings)
	}

	return rationale.String()
}

func evidenceFor(
	score *dispatchcandidateservice.CandidateScore,
	move *repositories.BoardMove,
) []agent.EvidenceRef {
	evidence := make([]agent.EvidenceRef, 0, 3)

	evidence = append(evidence,
		agent.EvidenceRef{
			Type: evidenceTypeMove,
			ID:   move.MoveID.String(),
			Note: fmt.Sprintf(
				"%s → %s, pickup window opens %d",
				originLabel(move),
				destinationLabel(move),
				move.OriginWindowStart,
			),
		},
		agent.EvidenceRef{
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

func moveStubFor(planned *portservices.DispatchPlannedAssignment) *repositories.BoardMove {
	return &repositories.BoardMove{
		MoveID:    planned.MoveID,
		ProNumber: planned.ProNumber,
	}
}
