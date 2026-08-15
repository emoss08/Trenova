package dispatchplanner

import "math"

const Unassigned = -1

const Forbidden = math.MaxFloat64 / 1e6

type Oracle interface {
	Cost(task, resource int) float64
	Commit(task, resource int)
}

type Options struct {
	MaxPerResource int
}

type Assignment struct {
	Task     int
	Resource int
	Sequence int
	Cost     float64
}

type Result struct {
	Assignments []Assignment
	Unassigned  []int
	TotalCost   float64
}

type SolveParams struct {
	Tasks     int
	Resources int
	Oracle    Oracle
	Options   Options
}

func Solve(params SolveParams) Result {
	if params.Tasks <= 0 || params.Resources <= 0 || params.Oracle == nil {
		unassigned := make([]int, 0, max(params.Tasks, 0))
		for task := range params.Tasks {
			unassigned = append(unassigned, task)
		}
		return Result{Assignments: make([]Assignment, 0), Unassigned: unassigned}
	}

	pending := make([]int, params.Tasks)
	for task := range pending {
		pending[task] = task
	}

	return insertGreedy(insertParams{
		Pending:   pending,
		Resources: params.Resources,
		Oracle:    params.Oracle,
		Options:   params.Options,
		State:     newSearchState(params.Resources),
	})
}

type searchState struct {
	committed []int
	sequence  []int
}

func newSearchState(resources int) *searchState {
	return &searchState{
		committed: make([]int, resources),
		sequence:  make([]int, resources),
	}
}

func (s *searchState) apply(assignments []Assignment) {
	for _, assignment := range assignments {
		if assignment.Resource < 0 || assignment.Resource >= len(s.committed) {
			continue
		}
		s.committed[assignment.Resource]++
		s.sequence[assignment.Resource]++
	}
}

type insertParams struct {
	Pending   []int
	Resources int
	Oracle    Oracle
	Options   Options
	State     *searchState
}

type chooser func(
	cost [][]float64,
	placed []bool,
	committed []int,
	opts Options,
) (index, resource int, best float64)

func insertGreedy(params insertParams) Result {
	return insertWith(params, cheapest)
}

func insertRegret(params insertParams) Result {
	return insertWith(params, mostRegret)
}

func insertWith(params insertParams, choose chooser) Result {
	result := Result{
		Assignments: make([]Assignment, 0, len(params.Pending)),
		Unassigned:  make([]int, 0, len(params.Pending)),
	}

	cost := make([][]float64, len(params.Pending))
	for index, task := range params.Pending {
		cost[index] = make([]float64, params.Resources)
		for resource := range cost[index] {
			cost[index][resource] = params.Oracle.Cost(task, resource)
		}
	}

	placed := make([]bool, len(params.Pending))

	for {
		index, resource, taskCost := choose(cost, placed, params.State.committed, params.Options)
		if index == Unassigned {
			break
		}

		task := params.Pending[index]
		params.Oracle.Commit(task, resource)

		placed[index] = true
		params.State.committed[resource]++
		result.Assignments = append(result.Assignments, Assignment{
			Task:     task,
			Resource: resource,
			Sequence: params.State.sequence[resource],
			Cost:     taskCost,
		})
		params.State.sequence[resource]++
		result.TotalCost += taskCost

		for remaining := range params.Pending {
			if placed[remaining] {
				continue
			}
			cost[remaining][resource] = params.Oracle.Cost(params.Pending[remaining], resource)
		}
	}

	for index, task := range params.Pending {
		if !placed[index] {
			result.Unassigned = append(result.Unassigned, task)
		}
	}

	return result
}

func cheapest(
	cost [][]float64,
	placed []bool,
	committed []int,
	opts Options,
) (index, resource int, best float64) {
	index, resource, best = Unassigned, Unassigned, Forbidden

	for candidateIndex := range cost {
		if placed[candidateIndex] {
			continue
		}
		for candidateResource, candidateCost := range cost[candidateIndex] {
			if opts.MaxPerResource > 0 && committed[candidateResource] >= opts.MaxPerResource {
				continue
			}
			if candidateCost >= Forbidden {
				continue
			}
			if index == Unassigned || candidateCost < best {
				index = candidateIndex
				resource = candidateResource
				best = candidateCost
			}
		}
	}

	return index, resource, best
}

func mostRegret(
	cost [][]float64,
	placed []bool,
	committed []int,
	opts Options,
) (index, resource int, best float64) {
	index, resource, best = Unassigned, Unassigned, Forbidden
	highest := math.Inf(-1)

	for candidateIndex := range cost {
		if placed[candidateIndex] {
			continue
		}

		firstCost, firstResource, secondCost := Forbidden, Unassigned, Forbidden
		for candidateResource, candidateCost := range cost[candidateIndex] {
			if opts.MaxPerResource > 0 && committed[candidateResource] >= opts.MaxPerResource {
				continue
			}
			if candidateCost >= Forbidden {
				continue
			}
			if firstResource == Unassigned || candidateCost < firstCost {
				secondCost = firstCost
				firstCost, firstResource = candidateCost, candidateResource
				continue
			}
			if candidateCost < secondCost {
				secondCost = candidateCost
			}
		}

		if firstResource == Unassigned {
			continue
		}

		regret := math.Inf(1)
		if secondCost < Forbidden {
			regret = secondCost - firstCost
		}

		if regret > highest {
			highest = regret
			index, resource, best = candidateIndex, firstResource, firstCost
		}
	}

	return index, resource, best
}
