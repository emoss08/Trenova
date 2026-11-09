package agentreflectionservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Of the first 40 memories look backs kept, about 35 described a tool call that
failed: a wrong argument, a field the tool does not take, an id where a code
was wanted. Each was a defect in the tool that every organization would learn
again, and stayed after the tool was fixed. The prompt rules out anything about
calling a tool, and keeps procedures for how a person showed the work is done.
*/
func TestSystemPrompt_KeepsNothingAboutHowToCallATool(t *testing.T) {
	t.Parallel()

	prompt := systemPrompt(&promptInput{HasPerson: true})

	assert.Contains(t, prompt, "anything about how to call a tool")
	assert.Contains(t, prompt, "A tool that was refused is a defect Trenova fixes in the tool")
	assert.Contains(t, prompt, "how this organization does a task, as a person showed it")
	assert.NotContains(t, prompt, "a tool that failed until its input changed")
	assert.NotContains(t, prompt, "toolName")
}

func TestOutputSchema_HasNoToolForALessonToBeAbout(t *testing.T) {
	t.Parallel()

	for _, hasPerson := range []bool{true, false} {
		schema := outputSchema(hasPerson)
		lessons := schema["properties"].(map[string]any)["lessons"].(map[string]any)
		item := lessons["items"].(map[string]any)

		assert.NotContains(t, item["properties"], "toolName")
		assert.NotContains(t, item["required"], "toolName")
	}
}
