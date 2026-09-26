package modeladapter

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func subsetTool() []ToolSpec {
	return []ToolSpec{{
		Name:        "transfer_to_billing",
		Description: "Transfer shipments to billing",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"shipmentIds": toolschema.RecordSubset("shipment", map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				}),
			},
			"required": []string{"shipmentIds"},
		},
	}}
}

// A tool's schema can carry keywords that are ours rather than JSON Schema's:
// the record-subset marker tells a person's approval form which ids it may
// untick. A strict endpoint (Gemini's OpenAI-compatible one among them)
// refuses a keyword it does not know, so no protocol ever sends one.
func TestAdapters_SendNoExtensionKeywordsToAModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    aiprovider.Kind
		adapter Adapter
		reply   any
	}{
		{
			"anthropic", aiprovider.KindAnthropicMessages, NewAnthropicAdapter(),
			map[string]any{
				"model":   "m",
				"content": []map[string]any{{"type": "text", "text": "hi"}},
			},
		},
		{
			"openai chat", aiprovider.KindOpenAIChat, NewOpenAIChatAdapter(),
			map[string]any{
				"model": "m",
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "hi"}},
				},
			},
		},
		{
			"openai responses", aiprovider.KindOpenAIResponses, NewOpenAIResponsesAdapter(),
			map[string]any{
				"model":  "m",
				"status": "completed",
				"output": []map[string]any{{
					"type":    "message",
					"role":    "assistant",
					"content": []map[string]any{{"type": "output_text", "text": "hi"}},
				}},
			},
		},
		{
			"ollama", aiprovider.KindOllama, NewOllamaAdapter(),
			map[string]any{
				"model":   "m",
				"message": map[string]any{"role": "assistant", "content": "hi"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, captured := captureServer(t, tc.reply)

			_, err := tc.adapter.Complete(t.Context(), callFor(tc.kind, server.URL, &Request{
				System:   "sys",
				Messages: UserMessage("hello"),
				Tools:    subsetTool(),
			}))
			require.NoError(t, err)

			encoded, err := sonic.Marshal((*captured)["tools"])
			require.NoError(t, err)
			assert.Contains(t, string(encoded), "shipmentIds")
			assert.NotContains(t, string(encoded), toolschema.KeySubsetOf)
		})
	}

	original := subsetTool()[0].Parameters["properties"].(map[string]any)["shipmentIds"]
	assert.Contains(t, original, toolschema.KeySubsetOf)
}
