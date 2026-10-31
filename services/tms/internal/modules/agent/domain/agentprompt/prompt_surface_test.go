package agentprompt_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_SaysThePersonIsInTheDeskWithWhatItIsFor(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Surface = agentpagecontext.SurfaceDesk
	rc.SurfaceGuide = &agentturn.RuntimePage{
		Name:    "Desk",
		Summary: "The Desk is where you talk to Trenova's AI agents.",
	}

	prompt := agentprompt.BuildSystemPrompt(definitionWithInstructions("Be brief."), rc)

	assert.Contains(t, prompt, "Where the person is talking to you: the Desk")
	assert.Contains(t, prompt, "They are in the Desk now, not on another page of Trenova.")
	assert.Contains(
		t,
		prompt,
		"What the Desk is for:\nThe Desk is where you talk to Trenova's AI agents.",
	)
	assert.Contains(t, prompt, "The page of Trenova the person had open before the Desk:")
	assert.NotContains(t, prompt, "What the person is looking at right now:")
}

func TestBuildSystemPrompt_NamesTheDeskWithoutItsGuide(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Surface = agentpagecontext.SurfaceDesk

	prompt := agentprompt.BuildSystemPrompt(definitionWithInstructions("Be brief."), rc)

	assert.Contains(t, prompt, "Where the person is talking to you: the Desk")
	assert.NotContains(t, prompt, "What the Desk is for:")
}

func TestBuildSystemPrompt_SaysThePersonIsInTheAssistantOverAPage(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Surface = agentpagecontext.SurfaceAssistant

	prompt := agentprompt.BuildSystemPrompt(definitionWithInstructions("Be brief."), rc)

	assert.Contains(t, prompt, "Where the person is talking to you: the assistant panel")
	assert.Contains(t, prompt, "What the person is looking at right now:")
	assert.NotContains(t, prompt, "had open before the Desk")
}

func TestBuildSystemPrompt_SaysNothingOfASurfaceTheClientDidNotName(t *testing.T) {
	t.Parallel()

	prompt := agentprompt.BuildSystemPrompt(definitionWithInstructions("Be brief."), fullContext())

	assert.NotContains(t, prompt, "Where the person is talking to you")
	assert.Contains(t, prompt, "What the person is looking at right now:")
}

func TestBuildSystemPrompt_RunsThePageTableToAnswerAboutItsRows(t *testing.T) {
	t.Parallel()

	rc := fullContext()
	rc.Page.View = &agentpagecontext.PageView{Resource: "agentRun", Query: "harbor"}
	rc.Surface = agentpagecontext.SurfaceAssistant

	prompt := agentprompt.BuildSystemPrompt(definitionWithInstructions("Be brief."), rc)

	assert.Contains(t, prompt, "To answer about these rows, call the matching list tool")
}
