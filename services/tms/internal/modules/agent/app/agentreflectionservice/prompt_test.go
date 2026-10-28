package agentreflectionservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
show_board refused five figures with "needs 1 to 4 figures" and the agent
proposed keeping that as a procedure. The refusal says it on every call, so the
lesson only repeats the tool, and goes stale when the tool changes. The prompt
rules it out and keeps procedures for what no tool said.
*/
func TestSystemPrompt_LeavesOutLessonsAToolsOwnFeedbackAlreadyGives(t *testing.T) {
	t.Parallel()

	prompt := systemPrompt(&promptInput{HasPerson: true})

	assert.Contains(t, prompt, "what a tool's own refusal, schema or note tells the agent "+
		"every time it is called")
	assert.Contains(t, prompt, "when the agent had to find them out and no tool told it")
}
