package llmports

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeAnswer_ReadsTheShapeTheTaskAskedFor(t *testing.T) {
	t.Parallel()

	var answer struct {
		Verdict string `json:"verdict"`
	}
	require.NoError(t, DecodeAnswer(aitask.TaskBillingDiagnosis, `{"verdict":"Agrees"}`, &answer))
	assert.Equal(t, "Agrees", answer.Verdict)

	err := DecodeAnswer(aitask.TaskBillingDiagnosis, "Sure! Here it is.", &answer)
	require.ErrorIs(t, err, ErrOutsideSchema)
	assert.Contains(t, err.Error(), string(aitask.TaskBillingDiagnosis))

	assert.NoError(t, DecodeAnswer(aitask.TaskBillingDiagnosis, "anything", nil), "no shape asked")
}

func TestChatTask_IsTheAssistantChatUnlessTheRequestNamesOne(t *testing.T) {
	t.Parallel()

	assert.Equal(t, aitask.TaskAssistantChat, (&ChatCompletionRequest{}).ChatTask())
	assert.Equal(t, aitask.TaskPaletteAnswer, (&ChatCompletionRequest{Task: aitask.TaskPaletteAnswer}).ChatTask())
}
