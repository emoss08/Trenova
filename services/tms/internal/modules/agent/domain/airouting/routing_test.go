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
		Tasks: []aitask.Task{aitask.TaskAssistantChat, aitask.TaskDailyBriefing},
	}
	ledger := &aiprovider.Provider{
		ID: pulid.MustNew(
			"aiprv_",
		), Name: "Ledger", Enabled: true, Trusted: true, Priority: 20, CreatedAt: 2,
		Tasks: []aitask.Task{aitask.TaskBillingDiagnosis},
	}
	draft := &aiprovider.Provider{
		Name:     "New one",
		Enabled:  true,
		Priority: 5,
		Tasks:    []aitask.Task{aitask.TaskAssistantChat, aitask.TaskBillingDiagnosis},
	}

	routes := PreviewRoutes([]*aiprovider.Provider{primary, ledger}, draft)

	chat := routeOf(t, routes, aitask.TaskAssistantChat)
	assert.Equal(t, "Primary", chat.Before.Name)
	assert.True(t, chat.After.Draft)
	assert.True(t, chat.Changed())

	briefing := routeOf(t, routes, aitask.TaskDailyBriefing)
	assert.Equal(t, primary.ID, briefing.After.ProviderID)
	assert.False(t, briefing.Changed())

	mapping := routeOf(t, routes, aitask.TaskBillingDiagnosis)
	assert.Equal(t, "Ledger", mapping.After.Name, "an untrusted provider cannot take a ledger task")

	extraction := routeOf(t, routes, aitask.TaskDocumentExtraction)
	assert.Nil(t, extraction.Before)
	assert.Nil(t, extraction.After)
	assert.False(t, extraction.Changed())
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
		Tasks: []aitask.Task{aitask.TaskAssistantChat, aitask.TaskFormulaAssistant},
	}
	off := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Off", Enabled: false, Priority: 15,
		Tasks: []aitask.Task{aitask.TaskAssistantChat},
	}
	target := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "AWS Bedrock", Enabled: true, Priority: 20,
		Tasks: []aitask.Task{aitask.TaskDailyBriefing},
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
	untrusted := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "vLLM", Enabled: true, Priority: 20,
		Tasks: []aitask.Task{aitask.TaskAssistantChat, aitask.TaskDailyBriefing},
	}
	disabled := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Off", Enabled: false, Priority: 30,
		Tasks: []aitask.Task{aitask.TaskDailyBriefing},
	}
	ordered := []*aiprovider.Provider{only, untrusted, disabled}

	_, err := PlanRouteTo(ordered, aitask.TaskBillingDiagnosis, untrusted.ID)
	var refusal *RouteRefusal
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Reason, "trusted")

	_, err = PlanRouteTo(ordered, aitask.TaskDailyBriefing, disabled.ID)
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, "provider is disabled", refusal.Reason)

	_, err = PlanRouteTo(ordered, aitask.TaskAssistantChat, untrusted.ID)
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Reason, "Groq takes no other task")

	_, err = PlanRouteTo(ordered, aitask.TaskAssistantChat, only.ID)
	require.ErrorIs(t, err, ErrRouteUnchanged)

	_, err = PlanRouteTo(ordered, aitask.TaskAssistantChat, pulid.MustNew("aiprv_"))
	require.ErrorIs(t, err, ErrRouteTargetMissing)
}
