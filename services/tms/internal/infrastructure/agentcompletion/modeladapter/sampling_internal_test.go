package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
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

	for _, task := range aitask.AllTasks() {
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

	classification := *SamplingForTask(aitask.TaskScopeClassification).Temperature
	extraction := *SamplingForTask(aitask.TaskDocumentExtraction).Temperature
	chat := *SamplingForTask(aitask.TaskAssistantChat).Temperature
	prose := *SamplingForTask(aitask.TaskDailyBriefing).Temperature

	assert.Zero(t, classification, "a yes-or-no should not move between two identical questions")
	assert.Less(t, extraction, chat, "a copied value is not a judgement call")
	assert.Less(t, chat, prose, "a turn that names tools is not a turn that writes prose")
}
