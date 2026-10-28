package agentruntime

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedDisclosures struct {
	mu          sync.Mutex
	preselected []agenttoolports.ToolDisclosure
	found       []agenttoolports.ToolDisclosure
}

func (r *recordedDisclosures) Preselected(_ context.Context, d *agenttoolports.ToolDisclosure) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.preselected = append(r.preselected, *d)
}

func (r *recordedDisclosures) Found(_ context.Context, d *agenttoolports.ToolDisclosure) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.found = append(r.found, *d)
}

func TestRun_ReportsTheOpeningRankingAndEachSearch(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(t,
		toolTurn(findToolsName, map[string]any{"need": "trailers due inspection"}),
		toolTurn("equipment_list_trailers", map[string]any{}),
		textTurn("Two trailers are due."),
	)
	observed := &recordedDisclosures{}
	service.disclosures = observed
	threadID := pulid.MustNew("athr_")

	_, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "say hello",
		ThreadID:   threadID,
	})
	require.NoError(t, err)

	require.Len(t, observed.preselected, 1)
	opening := observed.preselected[0]
	assert.True(t, opening.Disclosed)
	assert.Equal(t, threadID, opening.Attribution.ThreadID)
	assert.Len(t, opening.Ranked, preselectedTools)
	assert.Positive(t, opening.Held)
	for _, name := range opening.Ranked {
		assert.Contains(t, requestTools(completion.Requests[0]), name)
	}

	require.Len(t, observed.found, 1)
	search := observed.found[0]
	assert.Equal(t, "trailers due inspection", search.Need)
	assert.Contains(t, search.Ranked, "equipment_list_trailers")
	assert.Equal(t, threadID, search.Attribution.ThreadID)
}

func TestRun_SendsEveryHeldToolBelowTheConfiguredThreshold(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(t, textTurn("Hello."))
	everything := len(names) + 1
	service.threshold = &everything
	observed := &recordedDisclosures{}
	service.disclosures = observed

	_, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	assert.Empty(t, observed.preselected, "nothing was ranked: every tool was sent")
	assert.NotContains(t, requestTools(completion.Requests[0]), findToolsName)
	assert.Contains(t, requestTools(completion.Requests[0]), "equipment_list_trailers")
}
