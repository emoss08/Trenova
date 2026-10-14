package aianalyticsservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/pulid"
)

type taskInput struct {
	rows        []aianalyticsports.TaskCellRow
	ordered     []*aiprovider.Provider
	providers   providerLookup
	compareBy   aianalytics.CompareBy
	agentScoped bool
}

type taskDeployment struct {
	key       aianalyticsports.DeploymentKey
	runs      int
	completed int
}

func (d taskDeployment) completion() float64 {
	if d.runs == 0 {
		return 0
	}

	return float64(d.completed) / float64(d.runs)
}

type taskTally struct {
	task        aiprovider.Task
	runs        int
	cells       map[string]*aianalyticsports.TaskCell
	deployments []taskDeployment
}

func buildTasks(in *taskInput) []aianalyticsports.TaskRow {
	groupOf := groupKeyOf(in.compareBy)
	byTask := make(map[aiprovider.Task]*taskTally, 8)
	order := make([]aiprovider.Task, 0, 8)
	for _, row := range in.rows {
		entry, ok := byTask[row.Task]
		if !ok {
			entry = &taskTally{
				task:  row.Task,
				cells: make(map[string]*aianalyticsports.TaskCell, 8),
			}
			byTask[row.Task] = entry
			order = append(order, row.Task)
		}
		entry.runs += row.Runs
		key := groupOf(row.DeploymentKey)
		cell, ok := entry.cells[key]
		if !ok {
			cell = &aianalyticsports.TaskCell{SeriesKey: key}
			entry.cells[key] = cell
		}
		cell.Runs += row.Runs
		cell.Completed += row.Completed
		entry.deployments = append(entry.deployments, taskDeployment{
			key:       row.DeploymentKey,
			runs:      row.Runs,
			completed: row.Completed,
		})
	}

	enabled := make([]*aiprovider.Provider, 0, len(in.ordered))
	for _, provider := range in.ordered {
		if provider.Enabled {
			enabled = append(enabled, provider)
		}
	}
	providers := in.providers

	out := make([]aianalyticsports.TaskRow, 0, len(order))
	for _, task := range order {
		entry := byTask[task]
		row := aianalyticsports.TaskRow{Task: task, Cells: taskCells(entry.cells)}
		if best := bestCell(row.Cells); best != nil {
			row.BestSeriesKey = best.SeriesKey
		}
		current := aiprovider.RouteFor(enabled, task)
		if current != nil {
			ref := providers.ref(current.ID.String())
			row.Current = &ref
			row.CurrentSeriesKey = seriesKeyOf(current, in.compareBy)
		}
		if in.agentScoped {
			out = append(out, row)
			continue
		}
		row.Suggestion = suggestRoute(&routeQuestion{
			task:        task,
			current:     current,
			deployments: entry.deployments,
			ordered:     in.ordered,
			providers:   providers,
			groupOf:     groupOf,
		})
		out = append(out, row)
	}
	slices.SortFunc(out, func(left, right aianalyticsports.TaskRow) int {
		return cmp.Or(
			cmp.Compare(byTask[right.Task].runs, byTask[left.Task].runs),
			cmp.Compare(left.Task, right.Task),
		)
	})

	return out
}

func taskCells(cells map[string]*aianalyticsports.TaskCell) []aianalyticsports.TaskCell {
	out := make([]aianalyticsports.TaskCell, 0, len(cells))
	for _, cell := range cells {
		cell.CompletedRate = rate(cell.Completed, cell.Runs)
		cell.Few = !aianalytics.RanksByRuns(cell.Runs)
		out = append(out, *cell)
	}
	slices.SortFunc(out, func(left, right aianalyticsports.TaskCell) int {
		return cmp.Compare(left.SeriesKey, right.SeriesKey)
	})

	return out
}

func bestCell(cells []aianalyticsports.TaskCell) *aianalyticsports.TaskCell {
	var best *aianalyticsports.TaskCell
	ranked := 0
	for idx := range cells {
		cell := &cells[idx]
		if cell.Few || cell.CompletedRate == nil {
			continue
		}
		ranked++
		if best == nil || *cell.CompletedRate > *best.CompletedRate ||
			(*cell.CompletedRate == *best.CompletedRate && cell.Runs > best.Runs) {
			best = cell
		}
	}
	if ranked < 2 {
		return nil
	}

	return best
}

func seriesKeyOf(provider *aiprovider.Provider, compareBy aianalytics.CompareBy) string {
	if compareBy == aianalytics.CompareByProvider {
		return provider.ID.String()
	}

	return aianalytics.CanonicalModel(provider.Model)
}

type routeQuestion struct {
	task        aiprovider.Task
	current     *aiprovider.Provider
	deployments []taskDeployment
	ordered     []*aiprovider.Provider
	providers   providerLookup
	groupOf     keyFunc
}

func (q *routeQuestion) currentDeployment() (taskDeployment, bool) {
	var out taskDeployment
	for _, deployment := range q.deployments {
		if deployment.key.ProviderID != q.current.ID.String() {
			continue
		}
		out.key = deployment.key
		out.runs += deployment.runs
		out.completed += deployment.completed
	}

	return out, aianalytics.RanksByRuns(out.runs)
}

func suggestRoute(q *routeQuestion) *aianalyticsports.RouteSuggestion {
	if q.current == nil {
		return nil
	}
	current, ok := q.currentDeployment()
	if !ok {
		return nil
	}

	candidates := make([]taskDeployment, 0, len(q.deployments))
	for _, deployment := range q.deployments {
		if deployment.key.ProviderID == "" || deployment.key.ProviderID == q.current.ID.String() ||
			!aianalytics.RanksByRuns(deployment.runs) {
			continue
		}
		candidates = append(candidates, deployment)
	}
	slices.SortStableFunc(candidates, func(left, right taskDeployment) int {
		return cmp.Or(
			cmp.Compare(right.completion(), left.completion()),
			cmp.Compare(right.runs, left.runs),
		)
	})

	for _, candidate := range candidates {
		gain := (candidate.completion() - current.completion()) * percentScale
		if gain < aianalytics.MinRouteGainPoints {
			return nil
		}
		plan, err := aiprovider.PlanRouteTo(q.ordered, q.task, pulid.ID(candidate.key.ProviderID))
		if err != nil {
			continue
		}
		if aianalytics.CanonicalModel(plan.Target.Model) != candidate.key.ModelKey {
			continue
		}

		return &aianalyticsports.RouteSuggestion{
			Provider:   q.providers.ref(candidate.key.ProviderID),
			SeriesKey:  q.groupOf(candidate.key),
			GainPoints: gain,
		}
	}

	return nil
}
