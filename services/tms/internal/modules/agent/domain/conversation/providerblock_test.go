package conversation

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReasoningTrace_KeepsProviderBlocksThroughStorage(t *testing.T) {
	t.Parallel()

	trace := &ReasoningTrace{
		Text:         "plan",
		ProviderKind: "AnthropicMessages",
		Blocks: []ProviderBlock{
			{Type: "thinking", Text: "plan", Signature: "sig"},
			{Type: ProviderBlockServerToolUse, ID: "srvtoolu_1", Name: "advisor", Input: map[string]any{}},
			{
				Type: ProviderBlockAdvisorToolResult, ToolUseID: "srvtoolu_1", Model: "claude-opus-5-5",
				Result: &ServerToolResult{
					Type: "advisor_redacted_result", EncryptedContent: "enc", StopReason: "end_turn",
				},
			},
		},
	}

	encoded, err := sonic.Marshal(trace)
	require.NoError(t, err)
	decoded := &ReasoningTrace{}
	require.NoError(t, sonic.Unmarshal(encoded, decoded))
	assert.Equal(t, trace.Blocks[2], decoded.Blocks[2])
	assert.Equal(t, "claude-opus-5-5", decoded.AdvisorModel())

	var none *ReasoningTrace
	assert.Empty(t, none.AdvisorModel())
	assert.Empty(t, (&ReasoningTrace{Text: "only thinking"}).AdvisorModel())
}
