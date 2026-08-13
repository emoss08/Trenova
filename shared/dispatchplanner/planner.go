package dispatchplanner

import "math"

const Unassigned = -1

const Forbidden = math.MaxFloat64 / 1e6

// Oracle supplies assignment costs against resource state that advances as tasks
// are committed. Cost is re-queried for a resource after every Commit against it,
// so implementations are free to fold the committed task into the state they score
// from.
type Oracle interface {
	Cost(task, resource int) float64
	Commit(task, resource int)
}

type Options struct {
	// MaxPerResource caps how many tasks a single resource may accumulate. Zero
	// leaves the count unbounded.
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

// Solve greedily assigns tasks to resources, committing the globally cheapest
// feasible pairing on each round and re-costing only the resource that changed.
//
// Unlike a one-shot matching, a resource may take several tasks, and each
// committed task shifts the cost of every later task for that resource. That is
// what lets a sequence be built rather than a single snapshot matching.
func Solve(params SolveParams) Result {
	result := Result{
		Assignments: make([]Assignment, 0, params.Tasks),
		Unassigned:  make([]int, 0, params.Tasks),
	}

	if params.Tasks <= 0 || params.Resources <= 0 || params.Oracle == nil {
		for task := range params.Tasks {
			result.Unassigned = append(result.Unassigned, task)
		}
		return result
	}

	cost := make([][]float64, params.Tasks)
	for task := range cost {
		cost[task] = make([]float64, params.Resources)
		for resource := range cost[task] {
			cost[task][resource] = params.Oracle.Cost(task, resource)
		}
	}

	assigned := make([]bool, params.Tasks)
	committed := make([]int, params.Resources)
	sequence := make([]int, params.Resources)

	for {
		task, resource, taskCost := cheapest(cost, assigned, committed, params.Options)
		if task == Unassigned {
			break
		}

		params.Oracle.Commit(task, resource)

		assigned[task] = true
		committed[resource]++
		result.Assignments = append(result.Assignments, Assignment{
			Task:     task,
			Resource: resource,
			Sequence: sequence[resource],
			Cost:     taskCost,
		})
		sequence[resource]++
		result.TotalCost += taskCost

		for remaining := range params.Tasks {
			if assigned[remaining] {
				continue
			}
			cost[remaining][resource] = params.Oracle.Cost(remaining, resource)
		}
	}

	for task := range params.Tasks {
		if !assigned[task] {
			result.Unassigned = append(result.Unassigned, task)
		}
	}

	return result
}

func cheapest(
	cost [][]float64,
	assigned []bool,
	committed []int,
	opts Options,
) (task, resource int, best float64) {
	task, resource, best = Unassigned, Unassigned, Forbidden

	for candidateTask := range cost {
		if assigned[candidateTask] {
			continue
		}
		for candidateResource, candidateCost := range cost[candidateTask] {
			if opts.MaxPerResource > 0 && committed[candidateResource] >= opts.MaxPerResource {
				continue
			}
			if candidateCost >= Forbidden {
				continue
			}
			if task == Unassigned || candidateCost < best {
				task = candidateTask
				resource = candidateResource
				best = candidateCost
			}
		}
	}

	return task, resource, best
}
