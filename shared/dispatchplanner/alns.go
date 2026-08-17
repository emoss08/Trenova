package dispatchplanner

import (
	"math/rand"
	"sort"
)

const (
	minRuinFraction = 0.10
	maxRuinFraction = 0.40
	minRuinCount    = 2

	rewardImproved = 3.0
	rewardAccepted = 1.0
	operatorDecay  = 0.85
	minWeight      = 0.05
)

type ruinOperator int

const (
	ruinRandom ruinOperator = iota
	ruinWorstCost
	ruinResource
	ruinOperatorCount
)

type repairOperator int

const (
	repairGreedy repairOperator = iota
	repairRegret
	repairOperatorCount
)

type Rebuildable interface {
	Oracle

	Rebuild(assignments []Assignment) []Assignment
}

type ImproveParams struct {
	Resources  int
	Oracle     Rebuildable
	Options    Options
	Initial    Result
	Iterations int
	Seed       int64
}

func Improve(params ImproveParams) Result {
	best := cloneResult(params.Initial)

	if params.Oracle == nil || params.Iterations <= 0 || len(best.Assignments) == 0 {
		return best
	}

	rng := rand.New(rand.NewSource(params.Seed)) //nolint:gosec // plan reproducibility, not cryptography
	ruinWeights := newOperatorWeights(int(ruinOperatorCount))
	repairWeights := newOperatorWeights(int(repairOperatorCount))
	current := cloneResult(best)

	for range params.Iterations {
		ruinChoice := ruinOperator(ruinWeights.pick(rng))
		repairChoice := repairOperator(repairWeights.pick(rng))

		removed, kept := ruin(ruinChoice, current, rng)
		if len(removed) == 0 {
			ruinWeights.decay()
			repairWeights.decay()
			continue
		}

		candidate := repair(&repairParams{
			Kept:      kept,
			Removed:   removed,
			Unplaced:  current.Unassigned,
			Resources: params.Resources,
			Oracle:    params.Oracle,
			Options:   params.Options,
			Strategy:  repairChoice,
		})

		switch {
		case compare(candidate, best) > 0:
			best = cloneResult(candidate)
			current = candidate
			ruinWeights.reward(int(ruinChoice), rewardImproved)
			repairWeights.reward(int(repairChoice), rewardImproved)
		case compare(candidate, current) >= 0:
			current = candidate
			ruinWeights.reward(int(ruinChoice), rewardAccepted)
			repairWeights.reward(int(repairChoice), rewardAccepted)
		}

		ruinWeights.decay()
		repairWeights.decay()
	}

	best.Assignments = params.Oracle.Rebuild(best.Assignments)
	renumber(best.Assignments)
	best.TotalCost = totalCost(best.Assignments)

	return best
}

type repairParams struct {
	Kept      []Assignment
	Removed   []int
	Unplaced  []int
	Resources int
	Oracle    Rebuildable
	Options   Options
	Strategy  repairOperator
}

func repair(params *repairParams) Result {
	replayed := params.Oracle.Rebuild(params.Kept)

	state := newSearchState(params.Resources)
	state.apply(replayed)

	pending := make([]int, 0, len(params.Removed)+len(params.Unplaced))
	pending = append(pending, params.Removed...)
	pending = append(pending, params.Unplaced...)

	insertion := insertParams{
		Pending:   pending,
		Resources: params.Resources,
		Oracle:    params.Oracle,
		Options:   params.Options,
		State:     state,
	}

	var inserted Result
	switch params.Strategy {
	case repairRegret:
		inserted = insertRegret(insertion)
	case repairGreedy, repairOperatorCount:
		inserted = insertGreedy(insertion)
	default:
		inserted = insertGreedy(insertion)
	}

	assignments := make([]Assignment, 0, len(replayed)+len(inserted.Assignments))
	assignments = append(assignments, replayed...)
	assignments = append(assignments, inserted.Assignments...)
	renumber(assignments)

	return Result{
		Assignments: assignments,
		Unassigned:  inserted.Unassigned,
		TotalCost:   totalCost(assignments),
	}
}

func ruin(
	operator ruinOperator,
	current Result,
	rng *rand.Rand,
) (removed []int, kept []Assignment) {
	var drop map[int]struct{}

	switch operator {
	case ruinResource:
		drop = ruinOneResource(current, rng)
	case ruinWorstCost:
		drop = ruinCostliest(current, rng)
	case ruinRandom, ruinOperatorCount:
		drop = ruinAtRandom(current, rng)
	default:
		drop = ruinAtRandom(current, rng)
	}

	removed = make([]int, 0, len(drop))
	kept = make([]Assignment, 0, len(current.Assignments))

	for index, assignment := range current.Assignments {
		if _, ok := drop[index]; ok {
			removed = append(removed, assignment.Task)
			continue
		}
		kept = append(kept, assignment)
	}

	return removed, kept
}

func ruinAtRandom(current Result, rng *rand.Rand) map[int]struct{} {
	count := ruinCount(len(current.Assignments), rng)

	order := rng.Perm(len(current.Assignments))
	drop := make(map[int]struct{}, count)
	for _, index := range order[:count] {
		drop[index] = struct{}{}
	}

	return drop
}

func ruinCostliest(current Result, rng *rand.Rand) map[int]struct{} {
	count := ruinCount(len(current.Assignments), rng)

	order := make([]int, len(current.Assignments))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(i, j int) bool {
		return current.Assignments[order[i]].Cost > current.Assignments[order[j]].Cost
	})

	drop := make(map[int]struct{}, count)
	for _, index := range order[:count] {
		drop[index] = struct{}{}
	}

	return drop
}

func ruinOneResource(current Result, rng *rand.Rand) map[int]struct{} {
	byResource := make(map[int][]int, len(current.Assignments))
	order := make([]int, 0, len(current.Assignments))

	for index, assignment := range current.Assignments {
		if _, seen := byResource[assignment.Resource]; !seen {
			order = append(order, assignment.Resource)
		}
		byResource[assignment.Resource] = append(byResource[assignment.Resource], index)
	}

	if len(order) == 0 {
		return nil
	}

	chosen := byResource[order[rng.Intn(len(order))]]
	drop := make(map[int]struct{}, len(chosen))
	for _, index := range chosen {
		drop[index] = struct{}{}
	}

	return drop
}

func ruinCount(total int, rng *rand.Rand) int {
	if total <= minRuinCount {
		return total
	}

	fraction := minRuinFraction + rng.Float64()*(maxRuinFraction-minRuinFraction)

	return min(max(int(float64(total)*fraction), minRuinCount), total)
}

func compare(a, b Result) int {
	switch {
	case len(a.Assignments) > len(b.Assignments):
		return 1
	case len(a.Assignments) < len(b.Assignments):
		return -1
	case a.TotalCost < b.TotalCost:
		return 1
	case a.TotalCost > b.TotalCost:
		return -1
	default:
		return 0
	}
}

type operatorWeights struct {
	weights []float64
}

func newOperatorWeights(size int) *operatorWeights {
	weights := make([]float64, size)
	for i := range weights {
		weights[i] = 1
	}
	return &operatorWeights{weights: weights}
}

func (w *operatorWeights) pick(rng *rand.Rand) int {
	total := 0.0
	for _, weight := range w.weights {
		total += weight
	}
	if total <= 0 {
		return 0
	}

	roll := rng.Float64() * total
	for operator, weight := range w.weights {
		roll -= weight
		if roll <= 0 {
			return operator
		}
	}

	return len(w.weights) - 1
}

func (w *operatorWeights) reward(operator int, amount float64) {
	if operator < 0 || operator >= len(w.weights) {
		return
	}
	w.weights[operator] += amount
}

func (w *operatorWeights) decay() {
	for i := range w.weights {
		w.weights[i] = max(w.weights[i]*operatorDecay, minWeight)
	}
}

func renumber(assignments []Assignment) {
	next := make(map[int]int, len(assignments))
	for i := range assignments {
		resource := assignments[i].Resource
		assignments[i].Sequence = next[resource]
		next[resource]++
	}
}

func totalCost(assignments []Assignment) float64 {
	total := 0.0
	for _, assignment := range assignments {
		total += assignment.Cost
	}
	return total
}

func cloneResult(r Result) Result {
	assignments := make([]Assignment, len(r.Assignments))
	copy(assignments, r.Assignments)

	unassigned := make([]int, len(r.Unassigned))
	copy(unassigned, r.Unassigned)

	return Result{
		Assignments: assignments,
		Unassigned:  unassigned,
		TotalCost:   r.TotalCost,
	}
}
