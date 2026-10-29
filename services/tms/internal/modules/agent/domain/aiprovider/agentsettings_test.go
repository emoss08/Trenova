package aiprovider_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiproviderchange"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chatProvider() *aiprovider.Provider {
	return &aiprovider.Provider{
		Kind:  aiprovider.KindAnthropicMessages,
		Model: "claude-sonnet-5-5",
		Tasks: []aitask.Task{aitask.TaskAssistantChat},
	}
}

func agentSettingErrors(p *aiprovider.Provider) map[string]string {
	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)
	found := map[string]string{}
	for _, err := range multiErr.Errors {
		switch err.Field {
		case "advisorEnabled", "advisorModel", "scripts", "scriptWrites":
			found[err.Field] = err.Message
		}
	}

	return found
}

func TestAgentSettings_ShipOff(t *testing.T) {
	t.Parallel()

	provider := chatProvider()
	assert.Empty(t, provider.ResolvedAdvisorModel())
	assert.Equal(t, aiprovider.ScriptSettings{}, provider.ScriptSettings())
	assert.Empty(t, agentSettingErrors(provider))
}

func TestAgentSettings_TheAdvisorNeedsAPlatformThatServesIt(t *testing.T) {
	t.Parallel()

	for _, platform := range aiplatform.AllPlatforms() {
		provider := chatProvider()
		provider.Platform = platform
		provider.AdvisorEnabled = true
		errs := agentSettingErrors(provider)
		if platform.ServesAdvisor() {
			assert.Empty(t, errs, platform)
			assert.Equal(t, aiprovider.DefaultAdvisorModel, provider.ResolvedAdvisorModel())
			continue
		}
		assert.Contains(t, errs["advisorEnabled"], "Claude API or Claude Platform on AWS", platform)
		assert.Empty(t, provider.ResolvedAdvisorModel(), platform)
	}

	openai := chatProvider()
	openai.Kind = aiprovider.KindOpenAIResponses
	openai.AdvisorEnabled = true
	assert.NotEmpty(t, agentSettingErrors(openai)["advisorEnabled"])
	assert.ErrorIs(t, openai.CheckAdvisor(), aiprovider.ErrAdvisorNeedsAnthropic)
}

func TestAgentSettings_AnyAdvisorTheEndpointListsIsLeftToTheAPI(t *testing.T) {
	t.Parallel()

	provider := chatProvider()
	provider.AdvisorEnabled = true
	provider.AdvisorModel = " claude-haiku-5-5 "
	assert.Empty(t, agentSettingErrors(provider),
		"no model ranking: a weaker advisor is the API's to refuse")
	assert.Equal(t, "claude-haiku-5-5", provider.ResolvedAdvisorModel())
}

func TestAgentSettings_ARefusalHoldsUntilTheAdvisorOrModelChanges(t *testing.T) {
	t.Parallel()

	provider := chatProvider()
	provider.AdvisorEnabled = true
	provider.AdvisorRefusal = &aiprovider.AdvisorRefusal{
		Advisor: aiprovider.DefaultAdvisorModel,
		Model:   provider.Model,
		Message: "refused",
	}
	assert.Empty(t, provider.ResolvedAdvisorModel())

	other := *provider
	other.AdvisorModel = "claude-fable-5-1"
	assert.Equal(t, "claude-fable-5-1", other.ResolvedAdvisorModel())
	assert.True(t, aiproviderchange.AdvisorSettingsChanged(provider, &other))

	moved := *provider
	moved.Model = "claude-haiku-5-5"
	assert.Equal(t, aiprovider.DefaultAdvisorModel, moved.ResolvedAdvisorModel())
	assert.True(t, aiproviderchange.AdvisorSettingsChanged(provider, &moved))

	renamed := *provider
	renamed.Name = "Renamed"
	assert.False(t, aiproviderchange.AdvisorSettingsChanged(provider, &renamed))
}

func TestAgentSettings_ScriptWritesNeedScripts(t *testing.T) {
	t.Parallel()

	provider := chatProvider()
	provider.ScriptWrites = true
	require.Contains(t, agentSettingErrors(provider), "scriptWrites")
	assert.Equal(t, aiprovider.ScriptSettings{}, provider.ScriptSettings())

	provider.Scripts = true
	assert.Empty(t, agentSettingErrors(provider))
	assert.Equal(
		t,
		aiprovider.ScriptSettings{Scripts: true, Writes: true},
		provider.ScriptSettings(),
	)

	embedder := chatProvider()
	embedder.Tasks = []aitask.Task{aitask.TaskDocumentExtraction}
	embedder.Scripts = true
	assert.Contains(t, agentSettingErrors(embedder), "scripts")
}
