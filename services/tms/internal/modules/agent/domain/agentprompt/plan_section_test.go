package agentprompt_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/stretchr/testify/assert"
)

// A turn on a plan without Accounting & Cash is told so, in the part every
// turn shares, so it answers "that is part of the Accounting & Cash add-on"
// instead of saying Trenova cannot do it or guessing.
func TestBuildSystemPromptParts_NamesTheAddOnsThePlanLeavesOut(t *testing.T) {
	t.Parallel()

	d := definitionWithInstructions("Help the dispatch desk.")
	rc := fullContext()
	rc.NotOnPlan = []agentturn.PlanAddOn{
		{Name: "Accounting & Cash", Features: []string{"Accounting", "Factoring"}},
		{Name: "HR Suite", Features: []string{"Time off"}, MoreFeatures: 4},
	}

	parts := agentprompt.BuildSystemPromptParts(d, &rc)

	assert.Contains(t, parts.Stable, "## The organization's plan")
	assert.Contains(t, parts.Stable, "- Accounting & Cash (Accounting, Factoring)")
	assert.Contains(t, parts.Stable, "- HR Suite (Time off and 4 more)")
	assert.Contains(t, parts.Stable, "part of that add-on, which is not on the organization's plan")
	assert.NotContains(t, parts.Volatile, "Accounting & Cash")
}

// With every add-on on the prompt is exactly what it was.
func TestBuildSystemPromptParts_WithEveryAddOnOnAddsNothing(t *testing.T) {
	t.Parallel()

	d := definitionWithInstructions("Help the dispatch desk.")
	rc := fullContext()
	before := agentprompt.BuildSystemPromptParts(d, &rc)

	rc.NotOnPlan = []agentturn.PlanAddOn{}
	after := agentprompt.BuildSystemPromptParts(d, &rc)

	assert.Equal(t, before, after)
	assert.False(t, strings.Contains(after.Stable, "organization's plan\n"))
}
