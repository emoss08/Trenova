package agentprompt_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A provider caches the start of a prompt only while it stays the same bytes,
and the system prompt opened with the turn's own context — the page, the
record, the date, the memories — ahead of everything that never changes for
an agent. Every turn therefore re-read the whole prompt. What is the same on
every turn now comes first, and the turn's own part after it.
*/
func TestBuildSystemPromptParts_KeepsWhatEveryTurnSharesFirst(t *testing.T) {
	t.Parallel()

	d := definitionWithInstructions("Help the dispatch desk.")

	first := fullContext()
	second := fullContext()
	second.Now += 86400
	second.Page = &agentturn.PageContext{
		Path:       "/billing/invoices/inv_9",
		EntityType: "invoice",
		EntityID:   "inv_9",
	}
	second.Subject = &agentturn.RuntimeSubject{Type: agent.SubjectShipment, ID: "shp_2"}

	a := agentprompt.BuildSystemPromptParts(d, &first)
	b := agentprompt.BuildSystemPromptParts(d, &second)

	require.NotEmpty(t, a.Stable)
	require.NotEmpty(t, a.Volatile)
	assert.Equal(t, a.Stable, b.Stable, "two turns on different pages share the stable part")
	assert.NotEqual(t, a.Volatile, b.Volatile)
	assert.NotContains(t, a.Stable, "/shipments/shp_1", "the page is the turn's own")
	assert.Contains(t, a.Volatile, "/shipments/shp_1")
	assert.True(t, strings.HasPrefix(a.Stable, strings.SplitN(a.Stable, "\n", 2)[0]))

	whole := agentprompt.BuildSystemPrompt(d, first)
	assert.Equal(t, a.Stable+a.Volatile, whole, "the whole prompt is the two parts in order")
}

func TestBuildSystemPromptParts_PutsADisclosedToolSectionInTheStablePart(t *testing.T) {
	t.Parallel()

	d := definitionWithInstructions("Help the dispatch desk.")
	rc := fullContext()
	rc.ToolsDisclosed = true
	rc.Tools = []agentturn.ToolSummary{
		{Name: "shipment_get", Description: "Look up a shipment", Query: true, Loaded: true},
		{Name: "dispatch_assign_move", Description: "Assign a move", AlwaysProposes: true},
	}
	rc.ToolMap = []agentturn.ToolNamespace{
		{Name: "Dispatch", Covers: []string{"move"}, Tools: 2},
	}

	parts := agentprompt.BuildSystemPromptParts(d, &rc)

	assert.Contains(t, parts.Stable, "- Dispatch (2 tools): move")
	assert.Contains(t, parts.Stable, "dispatch_assign_move")
	assert.NotContains(t, parts.Volatile, "find_tools")
	assert.NotContains(t, parts.Volatile, "shipment_get")

	rc.Tools[0].Loaded, rc.Tools[1].Loaded = false, true
	other := agentprompt.BuildSystemPromptParts(d, &rc)
	assert.Equal(
		t,
		parts.Stable,
		other.Stable,
		"which tools a turn loaded does not move the prefix",
	)
}

func TestBuildSystemPromptParts_PutsThePersonsProfileInTheStablePart(t *testing.T) {
	t.Parallel()

	d := definitionWithInstructions("Help the dispatch desk.")
	rc := fullContext()
	rc.UserProfile = "Works the Acme Foods lanes out of Dallas."

	parts := agentprompt.BuildSystemPromptParts(d, &rc)

	assert.Contains(t, parts.Stable, "## About the person you are talking to")
	assert.Contains(t, parts.Stable, "Works the Acme Foods lanes out of Dallas.")
	assert.NotContains(t, parts.Volatile, "Acme Foods")
}
