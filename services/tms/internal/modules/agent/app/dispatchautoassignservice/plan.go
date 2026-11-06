package dispatchautoassignservice

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/timeutils"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/shopspring/decimal"
)

const (
	rationaleLines  = 3
	warningDiscount = 0.1
	millisPerMinute = 60_000
)

type plannedAssignmentParams struct {
	Pairing    *dispatchplanports.Pairing
	Tier       agent.AutonomyTier
	Threshold  decimal.Decimal
	ShadowMode bool
}

func plannedAssignmentFor(
	p *plannedAssignmentParams,
) *dispatchconsoleports.DispatchPlannedAssignment {
	confidence := confidenceFor(p.Pairing)

	planned := &dispatchconsoleports.DispatchPlannedAssignment{
		Pairing:    p.Pairing,
		Confidence: confidence,
		Rationale:  rationaleFor(p.Pairing),
	}
	planned.AutoExecutable = p.Tier == agent.TierAutoExecute &&
		!p.ShadowMode &&
		!p.Pairing.Candidate.Blocked() &&
		p.Pairing.Candidate.LocationKnown() &&
		confidence.GreaterThanOrEqual(p.Threshold)

	return planned
}

func confidenceFor(pairing *dispatchplanports.Pairing) decimal.Decimal {
	candidate := pairing.Candidate
	if candidate == nil || candidate.Blocked() || pairing.CoveragePriceCents <= 0 {
		return decimal.Zero
	}

	share := float64(pairing.PlanCostCents) / float64(pairing.CoveragePriceCents)
	confidence := 1 - share - warningDiscount*float64(candidate.Warnings())

	return decimal.NewFromFloat(min(max(confidence, 0), 1)).Round(2)
}

func rationaleFor(pairing *dispatchplanports.Pairing) string {
	candidate := pairing.Candidate
	if candidate == nil {
		return ""
	}

	var rationale strings.Builder
	fmt.Fprintf(
		&rationale,
		"%s costs the plan %s to cover %s, which is worth %s",
		candidate.WorkerName,
		dispatcheconomics.Dollars(pairing.PlanCostCents),
		pairing.ProNumber,
		dispatcheconomics.Dollars(pairing.CoveragePriceCents),
	)
	if lines := topLines(candidate.Cost.Lines); len(lines) > 0 {
		rationale.WriteString(": ")
		for i, line := range lines {
			if i > 0 {
				rationale.WriteString(", ")
			}
			fmt.Fprintf(
				&rationale,
				"%s %s",
				dispatcheconomics.Dollars(line.Cents),
				line.Component.Label(),
			)
		}
	}
	rationale.WriteString(".")
	if len(pairing.Alternatives) > 0 {
		next := pairing.Alternatives[0]
		fmt.Fprintf(
			&rationale,
			" Choosing %s instead adds %s to the plan.",
			next.WorkerName,
			dispatcheconomics.Dollars(next.ExtraCents),
		)
	}
	if warnings := candidate.Warnings(); warnings > 0 {
		fmt.Fprintf(&rationale, " %d advisory finding(s) to review.", warnings)
	}

	return rationale.String()
}

func topLines(lines []dispatcheconomics.Line) []dispatcheconomics.Line {
	positive := make([]dispatcheconomics.Line, 0, len(lines))
	for _, line := range lines {
		if line.Cents > 0 {
			positive = append(positive, line)
		}
	}
	slices.SortStableFunc(positive, func(a, b dispatcheconomics.Line) int {
		return cmp.Compare(b.Cents, a.Cents)
	})
	if len(positive) > rationaleLines {
		positive = positive[:rationaleLines]
	}

	return positive
}

func evidenceFor(pairing *dispatchplanports.Pairing) []agentevidence.EvidenceRef {
	candidate := pairing.Candidate
	evidence := make([]agentevidence.EvidenceRef, 0, 3)

	evidence = append(evidence,
		agentevidence.EvidenceRef{
			Type: evidenceTypeMove,
			ID:   pairing.MoveID.String(),
			Note: fmt.Sprintf(
				"%s → %s, pickup window opens %s; covering it is worth %s",
				pairing.OriginLabel,
				pairing.DestinationLabel,
				timeutils.FormatInstantUTC(pairing.PickupOpens),
				dispatcheconomics.Dollars(pairing.CoveragePriceCents),
			),
		},
		agentevidence.EvidenceRef{
			Type: evidenceTypeHOS,
			ID:   candidate.WorkerID.String(),
			Note: fmt.Sprintf(
				"drive %dm, shift %dm, cycle %dm remaining",
				candidate.DriveRemainingMs/millisPerMinute,
				candidate.ShiftRemainingMs/millisPerMinute,
				candidate.CycleRemainingMs/millisPerMinute,
			),
		})

	if candidate.DeadheadMiles != nil {
		evidence = append(evidence, agentevidence.EvidenceRef{
			Type: evidenceTypeDeadhead,
			ID:   candidate.TractorID.String(),
			Note: fmt.Sprintf(
				"%.0f empty miles to the pickup; %d minutes of appointment margin",
				*candidate.DeadheadMiles,
				candidate.MinutesOfSlack,
			),
		})
	}

	return evidence
}
