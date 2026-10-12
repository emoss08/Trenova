package aiprovider

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

// RouteChoice is the provider a task goes to. Draft marks the provider being
// edited, as the editor holds it, which has no ID until it is first saved.
type RouteChoice struct {
	ProviderID pulid.ID
	Name       string
	Draft      bool
}

// TaskRoute is where one task goes now and where it would go once a draft is
// saved. A nil choice is a task no provider serves.
type TaskRoute struct {
	Task   Task
	Before *RouteChoice
	After  *RouteChoice
}

// Changed reports whether saving the draft moves the task.
func (r *TaskRoute) Changed() bool {
	switch {
	case r.Before == nil || r.After == nil:
		return r.Before != r.After
	case r.After.Draft:
		return r.Before.ProviderID != r.After.ProviderID || r.Before.Name != r.After.Name
	default:
		return r.Before.ProviderID != r.After.ProviderID
	}
}

// RouteFor is the provider the router tries first for a task: the first, by
// priority and then age, that can serve it. Providers resting after repeated
// failures are skipped at run time and are not known here.
func RouteFor(providers []*Provider, task Task) *Provider {
	for _, provider := range providers {
		if ok, _ := provider.CanServeTask(task); ok {
			return provider
		}
	}
	return nil
}

// PreviewRoutes sets each task's provider now beside the one it would have
// with the draft saved. Enabled is the organization's enabled providers in
// routing order; the draft replaces the stored provider with its ID, or joins
// as a new one, last among those of its priority.
func PreviewRoutes(enabled []*Provider, draft *Provider) []TaskRoute {
	after := make([]*Provider, 0, len(enabled)+1)
	for _, provider := range enabled {
		if draft.ID.IsNotNil() && provider.ID == draft.ID {
			continue
		}
		after = append(after, provider)
	}
	if draft.Enabled {
		after = append(after, draft)
	}
	slices.SortStableFunc(after, func(a, b *Provider) int {
		if byPriority := cmp.Compare(a.Priority, b.Priority); byPriority != 0 {
			return byPriority
		}
		return compareAge(a, b, draft)
	})

	tasks := AllTasks()
	routes := make([]TaskRoute, 0, len(tasks))
	for _, task := range tasks {
		routes = append(routes, TaskRoute{
			Task:   task,
			Before: choiceOf(RouteFor(enabled, task), nil),
			After:  choiceOf(RouteFor(after, task), draft),
		})
	}
	return routes
}

// compareAge orders by when each was created; a draft not yet saved is the
// newest.
func compareAge(a, b, draft *Provider) int {
	newA := a == draft && a.ID.IsNil()
	newB := b == draft && b.ID.IsNil()
	switch {
	case newA && !newB:
		return 1
	case newB && !newA:
		return -1
	default:
		return cmp.Compare(a.CreatedAt, b.CreatedAt)
	}
}

func choiceOf(provider, draft *Provider) *RouteChoice {
	if provider == nil {
		return nil
	}
	return &RouteChoice{
		ProviderID: provider.ID,
		Name:       provider.Name,
		Draft:      draft != nil && provider == draft,
	}
}
