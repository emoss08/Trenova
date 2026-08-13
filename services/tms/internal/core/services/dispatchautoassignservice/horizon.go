package dispatchautoassignservice

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/emoss08/trenova/shared/pulid"
)

// horizonOracle scores one move against one driver at the driver's current
// projected state. Committing a pairing folds that move into the driver's
// commitments, so the next score for that driver departs from the move's
// destination and clock rather than from the driver's live position.
type horizonOracle struct {
	candidates *dispatchcandidateservice.Service
	moves      []*repositories.BoardMove
	drivers    []*repositories.BoardDriver
	snapshot   *dispatchcandidateservice.FleetSnapshot
	control    *dispatchcontrol.DispatchControl
	scores     [][]*dispatchcandidateservice.CandidateScore
}

func newHorizonOracle(
	p *solveParams,
	candidates *dispatchcandidateservice.Service,
) *horizonOracle {
	drivers := p.Snapshot.Drivers

	scores := make([][]*dispatchcandidateservice.CandidateScore, len(p.Moves))
	for i := range scores {
		scores[i] = make([]*dispatchcandidateservice.CandidateScore, len(drivers))
	}

	return &horizonOracle{
		candidates: candidates,
		moves:      p.Moves,
		drivers:    drivers,
		snapshot:   p.Snapshot,
		control:    p.Control,
		scores:     scores,
	}
}

func (o *horizonOracle) Cost(task, resource int) float64 {
	score := o.candidates.ScoreCandidate(&dispatchcandidateservice.ScoreCandidateRequest{
		Driver:   o.drivers[resource],
		Move:     o.moves[task],
		Snapshot: o.snapshot,
	})
	o.scores[task][resource] = score

	if cost := costFor(score, o.control); cost < assignmentsolver.Forbidden {
		return cost
	}

	return dispatchplanner.Forbidden
}

func (o *horizonOracle) Commit(task, resource int) {
	dispatchcandidateservice.CommitPlannedMove(
		&dispatchcandidateservice.CommitPlannedMoveRequest{
			Snapshot: o.snapshot,
			Move:     o.moves[task],
			Score:    o.scores[task][resource],
		},
	)
}

func (s *Service) solveHorizon(p *solveParams) *portservices.DispatchPlan {
	oracle := newHorizonOracle(p, s.candidates)

	result := dispatchplanner.Solve(dispatchplanner.SolveParams{
		Tasks:     len(p.Moves),
		Resources: len(p.Snapshot.Drivers),
		Oracle:    oracle,
		Options: dispatchplanner.Options{
			MaxPerResource: p.Control.HorizonMovesPerDriver(),
		},
	})

	return buildHorizonPlan(&buildHorizonPlanParams{
		Moves:        p.Moves,
		Result:       result,
		Oracle:       oracle,
		Control:      p.Control,
		AgentControl: p.AgentControl,
		Now:          p.Now,
	})
}

type buildHorizonPlanParams struct {
	Moves        []*repositories.BoardMove
	Result       dispatchplanner.Result
	Oracle       *horizonOracle
	Control      *dispatchcontrol.DispatchControl
	AgentControl *tenant.AgentControl
	Now          int64
}

func buildHorizonPlan(p *buildHorizonPlanParams) *portservices.DispatchPlan {
	tier := resolveTier(p.AgentControl)
	threshold := p.Control.ConfidenceThreshold()

	plan := &portservices.DispatchPlan{
		Assignments:  make([]*portservices.DispatchPlannedAssignment, 0, len(p.Result.Assignments)),
		Uncovered:    make([]*portservices.DispatchUncoveredMove, 0, len(p.Result.Unassigned)),
		Tours:        make([]*portservices.DispatchTour, 0, len(p.Oracle.drivers)),
		PlanningMode: dispatchcontrol.PlanningModeHorizon.String(),
		ShadowMode:   p.AgentControl.ShadowMode,
		AutonomyTier: tier,
		GeneratedAt:  p.Now,
	}

	tourByDriver := make(map[int]*portservices.DispatchTour, len(p.Oracle.drivers))
	tourOrder := make([]int, 0, len(p.Oracle.drivers))

	for _, assignment := range p.Result.Assignments {
		move := p.Moves[assignment.Task]
		score := p.Oracle.scores[assignment.Task][assignment.Resource]
		if score == nil {
			plan.Uncovered = append(plan.Uncovered, uncoveredFor(move, nil, p.Control))
			continue
		}

		tour, ok := tourByDriver[assignment.Resource]
		if !ok {
			tour = &portservices.DispatchTour{
				TourID:     pulid.MustNew("tour_"),
				WorkerID:   score.WorkerID,
				WorkerName: score.WorkerName,
				MoveIDs:    make([]pulid.ID, 0, 4),
				StartsAt:   score.ProjectedAvailable,
			}
			tourByDriver[assignment.Resource] = tour
			tourOrder = append(tourOrder, assignment.Resource)
		}

		planned := plannedAssignmentFor(&plannedAssignmentParams{
			Move:       move,
			Score:      score,
			Tier:       tier,
			Threshold:  threshold,
			ShadowMode: p.AgentControl.ShadowMode,
		})
		planned.TourID = tour.TourID
		planned.SequenceIndex = assignment.Sequence
		planned.ProjectedStartAt = score.ProjectedAvailable
		planned.ProjectedCompleteAt = dispatchcandidateservice.PlannedCompletion(move, score)
		planned.ProjectedDriveRemains = score.HOSProjectedDriveMs

		plan.Assignments = append(plan.Assignments, planned)
		plan.TotalScore += score.Score

		tour.MoveIDs = append(tour.MoveIDs, move.MoveID)
		tour.EndsAt = planned.ProjectedCompleteAt
		tour.TotalScore += score.Score
		if score.DeadheadMiles != nil {
			tour.TotalDeadheadMiles += *score.DeadheadMiles
		}
	}

	for _, index := range p.Result.Unassigned {
		plan.Uncovered = append(
			plan.Uncovered,
			uncoveredFor(p.Moves[index], p.Oracle.scores[index], p.Control),
		)
	}

	for _, driver := range tourOrder {
		plan.Tours = append(plan.Tours, tourByDriver[driver])
	}

	return plan
}
