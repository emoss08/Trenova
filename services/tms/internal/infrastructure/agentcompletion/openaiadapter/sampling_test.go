package openaiadapter

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySampling_LetsTheProviderOverrideTheTaskDefault(t *testing.T) {
	t.Parallel()

	stated := 0.85
	call := &modeladapter.Call{
		Provider: &aiprovider.Provider{ExtraBody: map[string]any{"temperature": stated}},
		Request: &modeladapter.Request{
			Sampling: modeladapter.SamplingForTask(aitask.TaskAssistantChat),
		},
	}

	body := chatRequest{}
	body.applySampling(call)

	// A value stated for a particular endpoint outranks a default chosen
	// for a class of work, so ours is left off and the merge carries theirs.
	assert.Nil(t, body.Temperature)
	require.NotNil(t, body.TopP, "top_p was not stated, so the default still fills it")
}

func TestApplySampling_FillsWhatTheProviderDidNotState(t *testing.T) {
	t.Parallel()

	call := &modeladapter.Call{
		Provider: &aiprovider.Provider{},
		Request: &modeladapter.Request{
			Sampling: modeladapter.SamplingForTask(aitask.TaskAssistantChat),
		},
	}

	body := chatRequest{}
	body.applySampling(call)

	require.NotNil(t, body.Temperature)
	require.NotNil(t, body.TopP)
}

// This adapter never talks to OpenAI — KindOpenAIChat has no default base
// URL and an optional credential precisely because it is the shape every
// other runtime exposes, and those read max_tokens. Sending OpenAI's newer
// spelling meant the ceiling was silently dropped on all of them.
func TestChatRequest_SendsTheCeilingEveryCompatibleServerReads(t *testing.T) {
	t.Parallel()

	body := chatRequest{MaxTokens: 4096}

	encoded, err := sonic.Marshal(body)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), `"max_tokens":4096`)
	assert.NotContains(t, string(encoded), "max_completion_tokens")
}

func TestReserveAnswerRoom_LeavesSomewhereToPutTheAnswer(t *testing.T) {
	t.Parallel()

	// With thinking on and a ceiling sized for an answer alone, the whole
	// budget goes on the chain of thought and the reply comes back empty —
	// which reads as a broken endpoint rather than as a budget too small.
	thinking := &modeladapter.Call{
		Provider: &aiprovider.Provider{ReasoningEffort: aireasoning.ReasoningMedium},
		Request:  &modeladapter.Request{},
	}
	floored := answerRoom(thinking, 1024)
	assert.Greater(t, floored, 1024)
	assert.GreaterOrEqual(t, floored, modeladapter.ReasoningAnswerFloor)

	// A model that does not think keeps the ceiling it was given.
	plain := &modeladapter.Call{Provider: &aiprovider.Provider{}, Request: &modeladapter.Request{}}
	assert.Equal(t, 1024, answerRoom(plain, 1024))
}
