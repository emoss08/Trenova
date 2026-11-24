package agentprompt_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/stretchr/testify/assert"
)

const softwareRuleStart = "- Do not write, review, explain, debug, or translate software"

func TestBuildSystemPrompt_DropsTheSoftwareRuleOnlyWithOpenTopics(t *testing.T) {
	t.Parallel()

	definition := &agentdefinition.Definition{Name: "Dispatch"}

	guarded := agentprompt.BuildSystemPrompt(definition, agentturn.RuntimeContext{})
	open := agentprompt.BuildSystemPrompt(definition, agentturn.RuntimeContext{OpenTopics: true})

	assert.Contains(t, guarded, softwareRuleStart)
	assert.NotContains(t, open, softwareRuleStart)
	assert.Contains(t, open, "## Boundaries")
	assert.Contains(t, open, "## Working with records")
}

func TestBuildSystemPrompt_SpeaksOfTransportationOnlyWithTheGuardOn(t *testing.T) {
	t.Parallel()

	definition := &agentdefinition.Definition{Name: "Dispatch"}

	guarded := agentprompt.BuildSystemPrompt(definition, agentturn.RuntimeContext{})
	open := agentprompt.BuildSystemPrompt(definition, agentturn.RuntimeContext{OpenTopics: true})

	assert.Contains(t, guarded, "You are Dispatch, an agent inside Trenova, "+
		"a transportation management system, working on behalf of one organization.")
	assert.Contains(t, guarded, agentprompt.DefaultPersona)

	assert.Contains(t, open, "You are Dispatch, an agent inside Trenova, "+
		"working on behalf of one organization.")
	assert.NotContains(t, open, "transportation management system, working on behalf")
	assert.NotContains(t, open, agentprompt.DefaultPersona)
	assert.Contains(t, open, "You are a helpful assistant. Be concise")
}
