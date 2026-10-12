package aiprovider

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func routeOf(t *testing.T, routes []TaskRoute, task Task) TaskRoute {
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

	primary := &Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Primary", Enabled: true, Priority: 10, CreatedAt: 1,
		Tasks: []Task{TaskAssistantChat, TaskDailyBriefing},
	}
	ledger := &Provider{
		ID: pulid.MustNew("aiprv_"), Name: "Ledger", Enabled: true, Trusted: true, Priority: 20, CreatedAt: 2,
		Tasks: []Task{TaskBillingDiagnosis},
	}
	draft := &Provider{Name: "New one", Enabled: true, Priority: 5, Tasks: []Task{TaskAssistantChat, TaskBillingDiagnosis}}

	routes := PreviewRoutes([]*Provider{primary, ledger}, draft)

	chat := routeOf(t, routes, TaskAssistantChat)
	assert.Equal(t, "Primary", chat.Before.Name)
	assert.True(t, chat.After.Draft)
	assert.True(t, chat.Changed())

	briefing := routeOf(t, routes, TaskDailyBriefing)
	assert.Equal(t, primary.ID, briefing.After.ProviderID)
	assert.False(t, briefing.Changed())

	mapping := routeOf(t, routes, TaskBillingDiagnosis)
	assert.Equal(t, "Ledger", mapping.After.Name, "an untrusted provider cannot take a ledger task")

	extraction := routeOf(t, routes, TaskDocumentExtraction)
	assert.Nil(t, extraction.Before)
	assert.Nil(t, extraction.After)
	assert.False(t, extraction.Changed())
}

func TestPreviewRoutesOfADisabledEditDropsTheProvider(t *testing.T) {
	t.Parallel()

	only := &Provider{ID: pulid.MustNew("aiprv_"), Name: "Only", Enabled: true, Tasks: []Task{TaskAssistantChat}}
	edited := *only
	edited.Enabled = false

	chat := routeOf(t, PreviewRoutes([]*Provider{only}, &edited), TaskAssistantChat)

	assert.Equal(t, "Only", chat.Before.Name)
	assert.Nil(t, chat.After)
	assert.True(t, chat.Changed())
}

func TestPreviewRoutesPutsANewDraftAfterProvidersOfItsPriority(t *testing.T) {
	t.Parallel()

	existing := &Provider{ID: pulid.MustNew("aiprv_"), Name: "Existing", Enabled: true, Priority: 1, CreatedAt: 99, Tasks: []Task{TaskAssistantChat}}
	draft := &Provider{Name: "Draft", Enabled: true, Priority: 1, Tasks: []Task{TaskAssistantChat}}

	chat := routeOf(t, PreviewRoutes([]*Provider{existing}, draft), TaskAssistantChat)

	assert.Equal(t, "Existing", chat.After.Name)
}
