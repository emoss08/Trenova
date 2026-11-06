package dispatchplanservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	horizonSearchSeed    = uint64(1)
	horizonBudgetPerMove = 400 * time.Microsecond
	horizonMinBudget     = 250 * time.Millisecond
	horizonMaxBudget     = 2 * time.Second
)

type horizonParams struct {
	Plan       *dispatchplanports.Plan
	Graph      *dispatchcandidateservice.Graph
	Snapshot   *dispatchcandidateservice.FleetSnapshot
	Candidates *dispatchcandidateservice.Service
	Control    *dispatchcontrol.DispatchControl
	Standing   map[pulid.ID]pulid.ID
}

func planHorizon(ctx context.Context, p *horizonParams) error {
	timeline := newHorizonTimeline(p)
	defer timeline.restore()

	moves := len(p.Graph.Moves)
	problem := problemOf(p.Graph)
	result, err := dispatchplanner.Solve(ctx, &dispatchplanner.Problem{
		Resources: len(p.Graph.Drivers),
		Arcs:      problem.Rows,
		Coverage:  problem.Uncovered,
		Opens:     opensOf(p.Graph.Moves),
		Successors: successorsOf(
			p.Graph.Moves,
			p.Snapshot.Model.Economics().CandidatesPerMove,
		),
		Incumbent: incumbentsOf(p.Graph, p.Standing),
		Timeline:  timeline,
		Options: dispatchplanner.Options{
			MaxPerResource: p.Control.HorizonMovesPerDriver(),
			Iterations:     p.Control.ResolvedHorizonSearchRounds() * moves,
			Budget:         horizonBudget(moves),
			RandomSeed:     horizonSearchSeed,
			StabilityCents: p.Snapshot.Model.Economics().ReplanChangeCents,
			Alternatives:   alternativesPerMove,
		},
	})
	if err != nil {
		return fmt.Errorf("plan horizon chains: %w", err)
	}

	writeHorizon(p, timeline, &result)

	return nil
}

func horizonBudget(moves int) time.Duration {
	return min(max(time.Duration(moves)*horizonBudgetPerMove, horizonMinBudget), horizonMaxBudget)
}

func incumbentsOf(
	graph *dispatchcandidateservice.Graph,
	standing map[pulid.ID]pulid.ID,
) []int {
	if len(standing) == 0 {
		return nil
	}

	drivers := make(map[pulid.ID]int, len(graph.Drivers))
	for i, driver := range graph.Drivers {
		drivers[driver.WorkerID] = i
	}
	incumbents := make([]int, len(graph.Moves))
	for i, move := range graph.Moves {
		incumbents[i] = dispatchplanner.Unassigned
		if workerID, ok := standing[move.MoveID]; ok {
			if driver, found := drivers[workerID]; found {
				incumbents[i] = driver
			}
		}
	}

	return incumbents
}

func writeHorizon(p *horizonParams, timeline *horizonTimeline, result *dispatchplanner.Result) {
	assignments := result.Assignments
	for start := 0; start < len(assignments); {
		end := start + 1
		for end < len(assignments) && assignments[end].Resource == assignments[start].Resource {
			end++
		}
		writeTour(p, timeline, result, assignments[start:end])
		start = end
	}

	for _, index := range result.Unassigned {
		addUncovered(p, index)
	}
}

func writeTour(
	p *horizonParams,
	timeline *horizonTimeline,
	result *dispatchplanner.Result,
	chain []dispatchplanner.Assignment,
) {
	driver := chain[0].Resource
	tasks := make([]int, len(chain))
	for i := range chain {
		tasks[i] = chain[i].Task
	}
	scores := timeline.replay(driver, tasks)

	var tour *dispatchplanports.Tour
	for i, assignment := range chain {
		score := scores[i]
		if score == nil {
			addUncovered(p, assignment.Task)
			continue
		}
		if tour == nil {
			tour = &dispatchplanports.Tour{
				TourID:     pulid.MustNew("tour_"),
				WorkerID:   score.WorkerID,
				WorkerName: score.WorkerName,
				MoveIDs:    make([]pulid.ID, 0, len(chain)),
				StartsAt:   score.ProjectedArrival,
			}
		}

		move := p.Graph.Moves[assignment.Task]
		planCost := score.HardCents + score.SoftCents - timeline.floor[assignment.Task]
		pairing := pairingOf(move, score, planCost, p.Graph.Coverage[assignment.Task])
		pairing.PreferenceFloorCents = timeline.floor[assignment.Task]
		pairing.TourID = tour.TourID
		pairing.SequenceIndex = len(tour.MoveIDs)
		pairing.DriverValueCents = result.ResourceValue[driver]
		pairing.Alternatives = horizonAlternatives(p, assignment.Alternatives)
		p.Plan.Pairings = append(p.Plan.Pairings, pairing)
		p.Plan.CostCents += planCost

		tour.MoveIDs = append(tour.MoveIDs, move.MoveID)
		tour.EndsAt = score.ProjectedFreeAt
		tour.CostCents += planCost
		if score.DeadheadMiles != nil {
			tour.EmptyMiles += *score.DeadheadMiles
		}
	}

	if tour != nil {
		p.Plan.Tours = append(p.Plan.Tours, tour)
	}
}

func addUncovered(p *horizonParams, index int) {
	uncovered := uncoveredOf(p.Graph, index)
	p.Plan.Uncovered = append(p.Plan.Uncovered, uncovered)
	p.Plan.UncoveredCents += uncovered.CoveragePriceCents
}

func horizonAlternatives(
	p *horizonParams,
	alternatives []dispatchplanner.Alternative,
) []dispatchplanprice.Alternative {
	out := make([]dispatchplanprice.Alternative, 0, len(alternatives))
	for _, alternative := range alternatives {
		driver := p.Graph.Drivers[alternative.Resource]
		out = append(out, dispatchplanprice.Alternative{
			WorkerID:   driver.WorkerID,
			WorkerName: p.Snapshot.DriverName(driver),
			ExtraCents: alternative.ExtraCost,
		})
	}

	return out
}
