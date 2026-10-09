package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_SaysThePersonIsInTheDeskWithWhatItIsFor(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Surface = agent.SurfaceDesk
	rc.SurfaceGuide = &agentdefinition.RuntimePage{
		Name:    "Desk",
		Summary: "The Desk is where you talk to Trenova's AI agents.",
	}

	prompt := definitionWithInstructions("Be brief.").BuildSystemPrompt(rc)

	assert.Contains(t, prompt, "Where the person is talking to you: the Desk")
	assert.Contains(t, prompt, "They are in the Desk now, not on another page of Trenova.")
	assert.Contains(t, prompt, "What the Desk is for:\nThe Desk is where you talk to Trenova's AI agents.")
	assert.Contains(t, prompt, "The page of Trenova the person had open before the Desk:")
	assert.NotContains(t, prompt, "What the person is looking at right now:")
}

func TestBuildSystemPrompt_NamesTheDeskWithoutItsGuide(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Surface = agent.SurfaceDesk

	prompt := definitionWithInstructions("Be brief.").BuildSystemPrompt(rc)

	assert.Contains(t, prompt, "Where the person is talking to you: the Desk")
	assert.NotContains(t, prompt, "What the Desk is for:")
}

func TestBuildSystemPrompt_SaysThePersonIsInTheAssistantOverAPage(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Surface = agent.SurfaceAssistant

	prompt := definitionWithInstructions("Be brief.").BuildSystemPrompt(rc)

	assert.Contains(t, prompt, "Where the person is talking to you: the assistant panel")
	assert.Contains(t, prompt, "What the person is looking at right now:")
	assert.NotContains(t, prompt, "had open before the Desk")
}

func TestBuildSystemPrompt_SaysNothingOfASurfaceTheClientDidNotName(t *testing.T) {
	t.Parallel()

	prompt := definitionWithInstructions("Be brief.").BuildSystemPrompt(fullContext())

	assert.NotContains(t, prompt, "Where the person is talking to you")
	assert.Contains(t, prompt, "What the person is looking at right now:")
}
