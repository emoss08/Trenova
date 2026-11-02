package airouting

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func routeOf(t *testing.T, routes []TaskRoute, task aitask.Task) TaskRoute {
	t.Helper()
	for _, route := range routes {
		if route.Task == task {
			return route
		}
	}
	require.Failf(t, "no route", "task %s", task)
	return TaskRoute{}
}

func TestPreviewRoutesMovesTasksToABetterPlacedDraft(t *testing.T) {
	t.Parallel()

	primary := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Primary", Enabled: true, Priority: 10, CreatedAt: 1,
		Tasks: []aitask.Task{aitask.TaskAssistantChat, aitask.TaskEvaluationJudge},
	}
	guard := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Guard", Enabled: true, Priority: 20, CreatedAt: 2,
		Tasks: []aitask.Task{aitask.TaskScopeClassification},
	}
	draft := &aiprovider.Provider{
		Name:     "New one",
		Enabled:  true,
		Priority: 5,
		Tasks:    []aitask.Task{aitask.TaskAssistantChat, aitask.TaskScopeClassification},
	}

	routes := PreviewRoutes([]*aiprovider.Provider{primary, guard}, draft)

	chat := routeOf(t, routes, aitask.TaskAssistantChat)
	assert.Equal(t, "Primary", chat.Before.Name)
	assert.True(t, chat.After.Draft)
	assert.True(t, chat.Changed())

	judge := routeOf(t, routes, aitask.TaskEvaluationJudge)
	assert.Equal(t, primary.ID, judge.After.ProviderID)
	assert.False(t, judge.Changed())

	scope := routeOf(t, routes, aitask.TaskScopeClassification)
	assert.Equal(t, "Guard", scope.Before.Name)
	assert.True(t, scope.After.Draft)

	general := routeOf(t, routes, aitask.TaskGeneral)
	assert.Nil(t, general.Before)
	assert.Nil(t, general.After)
	assert.False(t, general.Changed())
}

func TestPreviewRoutesListsOnlyRoutedTasks(t *testing.T) {
	t.Parallel()

	routes := PreviewRoutes(nil, &aiprovider.Provider{Name: "Draft", Enabled: true})

	tasks := make([]aitask.Task, 0, len(routes))
	for _, route := range routes {
		tasks = append(tasks, route.Task)
	}
	assert.Equal(
		t,
		aitask.RoutedTasks(),
		tasks,
		"Nova's tasks run on the Nova provider, not by route",
	)
}

func TestPreviewRoutesOfADisabledEditDropsTheProvider(t *testing.T) {
	t.Parallel()

	only := &aiprovider.Provider{
		ID:      pulid.MustNew("aiprv_"),
		Name:    "Only",
		Enabled: true,
		Tasks:   []aitask.Task{aitask.TaskAssistantChat},
	}
	edited := *only
	edited.Enabled = false

	chat := routeOf(
		t,
		PreviewRoutes([]*aiprovider.Provider{only}, &edited),
		aitask.TaskAssistantChat,
	)

	assert.Equal(t, "Only", chat.Before.Name)
	assert.Nil(t, chat.After)
	assert.True(t, chat.Changed())
}

func TestPreviewRoutesPutsANewDraftAfterProvidersOfItsPriority(t *testing.T) {
	t.Parallel()

	existing := &aiprovider.Provider{
		ID:        pulid.MustNew("aiprv_"),
		Name:      "Existing",
		Enabled:   true,
		Priority:  1,
		CreatedAt: 99,
		Tasks:     []aitask.Task{aitask.TaskAssistantChat},
	}
	draft := &aiprovider.Provider{
		Name:     "Draft",
		Enabled:  true,
		Priority: 1,
		Tasks:    []aitask.Task{aitask.TaskAssistantChat},
	}

	chat := routeOf(
		t,
		PreviewRoutes([]*aiprovider.Provider{existing}, draft),
		aitask.TaskAssistantChat,
	)

	assert.Equal(t, "Existing", chat.After.Name)
}

func TestPlanRouteToTakesTheTaskFromProvidersAheadAndGivesItToTheTarget(t *testing.T) {
	t.Parallel()

	first := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Workstation vLLM", Enabled: true, Priority: 10,
		Tasks: []aitask.Task{aitask.TaskAssistantChat, aitask.TaskScopeClassification},
	}
	off := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Off", Enabled: false, Priority: 15,
		Tasks: []aitask.Task{aitask.TaskAssistantChat},
	}
	target := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "AWS Bedrock", Enabled: true, Priority: 20,
		Tasks: []aitask.Task{aitask.TaskEvaluationJudge},
	}
	behind := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Groq", Enabled: true, Priority: 30,
		Tasks: []aitask.Task{aitask.TaskAssistantChat},
	}

	plan, err := PlanRouteTo(
		[]*aiprovider.Provider{first, off, target, behind},
		aitask.TaskAssistantChat,
		target.ID,
	)
	require.NoError(t, err)
	assert.Same(t, target, plan.Target)
	assert.True(t, plan.Assign)
	assert.Equal(t, []*aiprovider.Provider{first}, plan.Unassign)
}

func TestPlanRouteToRefusesWhatWouldBreakOrChangeNothing(t *testing.T) {
	t.Parallel()

	only := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Groq", Enabled: true, Priority: 10,
		Tasks: []aitask.Task{aitask.TaskAssistantChat},
	}
	second := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "vLLM", Enabled: true, Priority: 20,
		Tasks: []aitask.Task{aitask.TaskAssistantChat, aitask.TaskEvaluationJudge},
	}
	disabled := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Off", Enabled: false, Priority: 30,
		Tasks: []aitask.Task{aitask.TaskEvaluationJudge},
	}
	ordered := []*aiprovider.Provider{only, second, disabled}

	_, err := PlanRouteTo(ordered, aitask.TaskBillingDiagnosis, second.ID)
	var refusal *RouteRefusalError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Reason, "Nova provider", "Nova's tasks are never routed")

	_, err = PlanRouteTo(ordered, aitask.TaskEvaluationJudge, disabled.ID)
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, "provider is disabled", refusal.Reason)

	_, err = PlanRouteTo(ordered, aitask.TaskAssistantChat, second.ID)
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Reason, "Groq takes no other task")

	_, err = PlanRouteTo(ordered, aitask.TaskAssistantChat, only.ID)
	require.ErrorIs(t, err, ErrRouteUnchanged)

	_, err = PlanRouteTo(ordered, aitask.TaskAssistantChat, pulid.MustNew("aiprv_"))
	require.ErrorIs(t, err, ErrRouteTargetMissing)
}
