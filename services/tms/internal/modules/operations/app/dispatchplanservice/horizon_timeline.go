package dispatchplanservice

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/emoss08/trenova/shared/pulid"
)

const heldStale = int32(-2)

type horizonNode struct {
	parent     int32
	commitment *dispatchboardports.WorkerCommitment
}

type horizonTimeline struct {
	graph      *dispatchcandidateservice.Graph
	snapshot   *dispatchcandidateservice.FleetSnapshot
	candidates *dispatchcandidateservice.Service
	floor      []int64
	base       map[pulid.ID][]*dispatchboardports.WorkerCommitment
	nodes      []horizonNode
	held       map[int]int32
	path       []*dispatchboardports.WorkerCommitment
}

func newHorizonTimeline(p *horizonParams) *horizonTimeline {
	if p.Snapshot.CommitmentsByWorker == nil {
		p.Snapshot.CommitmentsByWorker = make(map[pulid.ID][]*dispatchboardports.WorkerCommitment)
	}

	t := &horizonTimeline{
		graph:      p.Graph,
		snapshot:   p.Snapshot,
		candidates: p.Candidates,
		floor:      make([]int64, len(p.Graph.Moves)),
		base:       maps.Clone(p.Snapshot.CommitmentsByWorker),
		held:       make(map[int]int32),
	}
	for i, arcs := range p.Graph.Arcs {
		for j, arc := range arcs {
			if soft := arc.Candidate.SoftCents; j == 0 || soft < t.floor[i] {
				t.floor[i] = soft
			}
		}
	}

	return t
}

func (t *horizonTimeline) Extend(ext dispatchplanner.Extension) dispatchplanner.Step {
	move := t.graph.Moves[ext.Task]

	var (
		score *dispatchplanports.CandidateScore
		cost  int64
	)
	if ext.Parent == dispatchplanner.Root {
		arc, ok := arcFor(t.graph.Arcs[ext.Task], ext.Resource)
		if !ok {
			return dispatchplanner.Step{}
		}
		score, cost = arc.Candidate, arc.CostCents
	} else {
		t.hold(ext.Resource, ext.Parent)
		score = t.score(ext.Resource, move)
		if score == nil {
			return dispatchplanner.Step{}
		}
		cost = score.HardCents + score.SoftCents - t.floor[ext.Task]
	}

	commitment := t.commitmentOf(move, score)
	if commitment == nil {
		return dispatchplanner.Step{}
	}
	for len(t.nodes) <= int(ext.Node) {
		t.nodes = append(t.nodes, horizonNode{})
	}
	t.nodes[ext.Node] = horizonNode{parent: ext.Parent, commitment: commitment}

	return dispatchplanner.Step{Cost: cost, Feasible: true}
}

func (t *horizonTimeline) Reset() {
	t.nodes = t.nodes[:0]
	for driver := range t.held {
		t.held[driver] = heldStale
	}
}

func (t *horizonTimeline) replay(driver int, tasks []int) []*dispatchplanports.CandidateScore {
	scores := make([]*dispatchplanports.CandidateScore, len(tasks))
	t.path = t.path[:0]
	defer func() { t.held[driver] = heldStale }()

	for i, task := range tasks {
		move := t.graph.Moves[task]
		var score *dispatchplanports.CandidateScore
		if i == 0 {
			arc, ok := arcFor(t.graph.Arcs[task], driver)
			if !ok {
				return scores
			}
			score = arc.Candidate
		} else {
			t.install(driver, t.path)
			if score = t.score(driver, move); score == nil {
				return scores
			}
		}

		commitment := t.commitmentOf(move, score)
		if commitment == nil {
			return scores
		}
		scores[i] = score
		t.path = append(t.path, commitment)
	}

	return scores
}

func (t *horizonTimeline) restore() {
	for driver := range t.held {
		workerID := t.graph.Drivers[driver].WorkerID
		if base, ok := t.base[workerID]; ok {
			t.snapshot.CommitmentsByWorker[workerID] = base
		} else {
			delete(t.snapshot.CommitmentsByWorker, workerID)
		}
		t.snapshot.RefreshProfile(workerID)
	}
	clear(t.held)
}

func (t *horizonTimeline) hold(driver int, node int32) {
	if held, ok := t.held[driver]; ok && held == node {
		return
	}

	t.path = t.path[:0]
	for n := node; n != dispatchplanner.Root; n = t.nodes[n].parent {
		t.path = append(t.path, t.nodes[n].commitment)
	}
	slices.Reverse(t.path)
	t.install(driver, t.path)
	t.held[driver] = node
}

func (t *horizonTimeline) install(
	driver int,
	planned []*dispatchboardports.WorkerCommitment,
) {
	workerID := t.graph.Drivers[driver].WorkerID
	base := t.base[workerID]
	commitments := make([]*dispatchboardports.WorkerCommitment, 0, len(base)+len(planned))
	commitments = append(commitments, base...)
	commitments = append(commitments, planned...)
	t.snapshot.CommitmentsByWorker[workerID] = commitments
	t.snapshot.RefreshProfile(workerID)
}

func (t *horizonTimeline) score(
	driver int,
	move *dispatchboardports.BoardMove,
) *dispatchplanports.CandidateScore {
	score := t.candidates.ScoreCandidate(&dispatchcandidateservice.ScoreCandidateRequest{
		Driver:   t.graph.Drivers[driver],
		Move:     move,
		Snapshot: t.snapshot,
	})
	if !dispatchcandidateservice.Admissible(t.snapshot, score) {
		return nil
	}

	return score
}

func (t *horizonTimeline) commitmentOf(
	move *dispatchboardports.BoardMove,
	score *dispatchplanports.CandidateScore,
) *dispatchboardports.WorkerCommitment {
	return dispatchcandidateservice.PlannedCommitment(move, score)
}
