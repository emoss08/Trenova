package anthropicadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_ReportsTheCeilingItSent(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
		"model":       "claude-x",
		"stop_reason": "end_turn",
		"content":     []any{map[string]any{"type": "text", "text": "Done."}},
	})

	resp, err := NewAnthropicAdapter().Complete(t.Context(), modeladaptertest.ReasoningCall(
		aiprovider.KindAnthropicMessages, server.URL, aireasoning.ReasoningMedium,
		&modeladapter.Request{Messages: modeladapter.UserMessage("go")},
	))

	require.NoError(t, err)
	assert.InDelta(t, 4096+modeladapter.ThinkingAnswerRoom, (*captured)["max_tokens"], 0)
	assert.Equal(t, 4096+modeladapter.ThinkingAnswerRoom, resp.OutputLimit)
}
