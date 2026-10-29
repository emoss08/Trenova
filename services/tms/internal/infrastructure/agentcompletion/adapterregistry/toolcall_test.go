package adapterregistry

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/anthropicadapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/openaiadapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every protocol must survive a turn with no tools, since most turns have none.
func TestAdapters_OmitToolFieldsWhenNoToolsOffered(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    aiprovider.Kind
		adapter modeladapter.Adapter
		reply   any
	}{
		{
			"anthropic", aiprovider.KindAnthropicMessages, anthropicadapter.NewAnthropicAdapter(),
			map[string]any{
				"model":   "m",
				"content": []map[string]any{{"type": "text", "text": "hi"}},
			},
		},
		{
			"openai chat", aiprovider.KindOpenAIChat, openaiadapter.NewOpenAIChatAdapter(),
			map[string]any{
				"model": "m",
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "hi"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, captured := modeladaptertest.CaptureServer(t, tc.reply)

			resp, err := tc.adapter.Complete(
				t.Context(),
				modeladaptertest.CallFor(tc.kind, server.URL, &modeladapter.Request{
					System:   "sys",
					Messages: modeladapter.UserMessage("hello"),
				}),
			)
			require.NoError(t, err)

			assert.Equal(t, "hi", resp.Text)
			assert.Empty(t, resp.ToolCalls)
			assert.NotContains(t, *captured, "tools")
		})
	}
}
