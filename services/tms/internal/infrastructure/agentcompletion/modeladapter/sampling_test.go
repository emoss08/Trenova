package modeladapter

import (
	"testing"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Nothing used to be sent, so every call ran at the endpoint's own
// defaults — typically temperature 1 with no nucleus cutoff, which samples
// from the whole vocabulary including its tail. A large model mostly
// survives that; a mixture-of-experts model with a few billion active
// parameters degenerates into fragments of other languages and invented
// markup, which is what a production transcript showed.
func TestSamplingForTask_AlwaysNamesBothKnobs(t *testing.T) {
	t.Parallel()

	for _, task := range aiprovider.AllTasks() {
		sampling := SamplingForTask(task)
		if !task.Generates() {
			assert.Nil(t, sampling.Temperature, "%s samples no tokens", task)
			assert.Nil(t, sampling.TopP, "%s samples no tokens", task)

			continue
		}
		require.NotNil(t, sampling.Temperature, task)
		require.NotNil(t, sampling.TopP, task)
		assert.Less(t, *sampling.TopP, 1.0, "%s: the tail is never the right token", task)
		assert.LessOrEqual(t, *sampling.Temperature, 1.0, task)
		assert.GreaterOrEqual(t, *sampling.Temperature, 0.0, task)
	}
}

func TestSamplingForTask_IsColderForWorkWithOneRightAnswer(t *testing.T) {
	t.Parallel()

	classification := *SamplingForTask(aiprovider.TaskScopeClassification).Temperature
	extraction := *SamplingForTask(aiprovider.TaskDocumentExtraction).Temperature
	chat := *SamplingForTask(aiprovider.TaskAssistantChat).Temperature
	prose := *SamplingForTask(aiprovider.TaskDailyBriefing).Temperature

	assert.Zero(t, classification, "a yes-or-no should not move between two identical questions")
	assert.Less(t, extraction, chat, "a copied value is not a judgement call")
	assert.Less(t, chat, prose, "a turn that names tools is not a turn that writes prose")
}

func TestApplySampling_LetsTheProviderOverrideTheTaskDefault(t *testing.T) {
	t.Parallel()

	stated := 0.85
	call := &Call{
		Provider: &aiprovider.Provider{ExtraBody: map[string]any{"temperature": stated}},
		Request:  &Request{Sampling: SamplingForTask(aiprovider.TaskAssistantChat)},
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

	call := &Call{
		Provider: &aiprovider.Provider{},
		Request:  &Request{Sampling: SamplingForTask(aiprovider.TaskAssistantChat)},
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
	thinking := &Call{
		Provider: &aiprovider.Provider{ReasoningEffort: aiprovider.ReasoningMedium},
		Request:  &Request{},
	}
	body := chatRequest{MaxTokens: 1024}
	body.reserveAnswerRoom(thinking)
	assert.Greater(t, body.MaxTokens, 1024)
	assert.GreaterOrEqual(t, body.MaxTokens, thinkingFloor+thinkingAnswerRoom)

	// A model that does not think keeps the ceiling it was given.
	plain := &Call{Provider: &aiprovider.Provider{}, Request: &Request{}}
	quiet := chatRequest{MaxTokens: 1024}
	quiet.reserveAnswerRoom(plain)
	assert.Equal(t, 1024, quiet.MaxTokens)
}
