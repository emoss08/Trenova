package dispatchplanservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/assignmentsolver"
)

const (
	alternativesPerMove  = 3
	alternativesExamined = 2 * alternativesPerMove
)

func problemOf(graph *dispatchcandidateservice.Graph) assignmentsolver.Problem {
	problem := assignmentsolver.Problem{
		Cols:      len(graph.Drivers),
		Rows:      make([][]assignmentsolver.Arc, len(graph.Moves)),
		Uncovered: graph.Coverage,
	}
	for i, arcs := range graph.Arcs {
		problem.Rows[i] = make([]assignmentsolver.Arc, 0, len(arcs))
		for _, arc := range arcs {
			problem.Rows[i] = append(problem.Rows[i], assignmentsolver.Arc{
				Col:  arc.Driver,
				Cost: arc.CostCents,
			})
		}
	}

	return problem
}

func planImmediate(
	plan *dispatchplanports.Plan,
	graph *dispatchcandidateservice.Graph,
	stability stability,
) {
	solution := assignmentsolver.SolveWithPrices(stability.apply(problemOf(graph)))

	for i, move := range graph.Moves {
		col := solution.RowAssignment[i]
		if col == assignmentsolver.Unassigned {
			uncovered := uncoveredOf(graph, i)
			plan.Uncovered = append(plan.Uncovered, uncovered)
			plan.UncoveredCents += uncovered.CoveragePriceCents
			continue
		}

		arc, _ := arcFor(graph.Arcs[i], col)
		pairing := pairingOf(move, arc.Candidate, graph.Coverage[i])
		pairing.DriverValueCents = solution.ColumnValue(col)
		pairing.Alternatives = alternatives(graph, solution, i, col)
		plan.Pairings = append(plan.Pairings, pairing)
		plan.CostCents += arc.Candidate.Cost.TotalCents
	}
}

func arcFor(arcs []dispatchcandidateservice.Arc, driver int) (dispatchcandidateservice.Arc, bool) {
	for _, arc := range arcs {
		if arc.Driver == driver {
			return arc, true
		}
	}

	return dispatchcandidateservice.Arc{}, false
}

func pairingOf(
	move *dispatchboardports.BoardMove,
	candidate *dispatchplanports.CandidateScore,
	coverage int64,
) *dispatchplanports.Pairing {
	return &dispatchplanports.Pairing{
		MoveID:      move.MoveID,
		MoveVersion: move.MoveVersion,
		ProNumber:   move.ProNumber,
		OriginLabel: placeLabel(move.OriginCity, move.OriginState, move.OriginName),
		DestinationLabel: placeLabel(
			move.DestinationCity,
			move.DestinationState,
			move.DestinationName,
		),
		PickupOpens:         move.OriginWindowStart,
		Candidate:           candidate,
		CoveragePriceCents:  coverage,
		ProjectedStartAt:    candidate.ProjectedArrival,
		ProjectedCompleteAt: candidate.ProjectedFreeAt,
	}
}

func placeLabel(city, state, name string) string {
	if city == "" {
		return name
	}
	if state == "" {
		return city
	}

	return city + ", " + state
}

func alternatives(
	graph *dispatchcandidateservice.Graph,
	solution assignmentsolver.Solution,
	row, chosen int,
) []dispatchplanprice.Alternative {
	out := make([]dispatchplanprice.Alternative, 0, alternativesExamined)
	for _, arc := range graph.Arcs[row] {
		if len(out) == alternativesExamined {
			break
		}
		if arc.Driver == chosen {
			continue
		}
		extra, ok := solution.OverrideCost(row, arc.Driver)
		if !ok {
			continue
		}
		out = append(out, dispatchplanprice.Alternative{
			WorkerID:   arc.Candidate.WorkerID,
			WorkerName: arc.Candidate.WorkerName,
			ExtraCents: extra,
			CostCents:  arc.Candidate.Cost.TotalCents,
			Lines:      arc.Candidate.Cost.Lines,
		})
	}
	slices.SortStableFunc(out, func(a, b dispatchplanprice.Alternative) int {
		return cmp.Compare(a.ExtraCents, b.ExtraCents)
	})

	return out[:min(len(out), alternativesPerMove)]
}

func uncoveredOf(graph *dispatchcandidateservice.Graph, row int) *dispatchplanports.Uncovered {
	move := graph.Moves[row]
	uncovered := &dispatchplanports.Uncovered{
		MoveID:             move.MoveID,
		ProNumber:          move.ProNumber,
		StartsAt:           move.OriginWindowStart,
		CoveragePriceCents: graph.Coverage[row],
	}

	arcs := graph.Arcs[row]
	if len(arcs) > 0 {
		cheapest := arcs[0]
		for _, arc := range arcs[1:] {
			if arc.CostCents < cheapest.CostCents {
				cheapest = arc
			}
		}
		uncovered.BestCandidate = cheapest.Candidate
		uncovered.Reason = dispatchplanprice.UncoveredOutbid
		if cheapest.CostCents >= graph.Coverage[row] {
			uncovered.Reason = dispatchplanprice.UncoveredNotWorthIt
		}
		return uncovered
	}

	rejected := graph.Rejected[row]
	uncovered.BestCandidate = rejected
	switch {
	case rejected == nil:
		uncovered.Reason = dispatchplanprice.UncoveredNoCandidate
	case rejected.Blocked():
		uncovered.Reason = dispatchplanprice.UncoveredDisqualified
	case !rejected.LocationKnown():
		uncovered.Reason = dispatchplanprice.UncoveredLocationUnknown
	default:
		uncovered.Reason = dispatchplanprice.UncoveredBeyondDeadhead
	}

	return uncovered
}
