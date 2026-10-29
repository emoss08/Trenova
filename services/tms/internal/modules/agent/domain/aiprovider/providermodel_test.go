package aiprovider

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aimodel"
	"github.com/stretchr/testify/assert"
)

/*
An operator serving a Qwen model with its context stretched to 128k, or a
Llama with it cut to 16k to fit a GPU, knows the window better than the id
does, and a window read off the id would compact the first conversation far
too late or far too soon.
*/
func TestContextWindow_TheConfiguredWindowWinsOverTheModelID(t *testing.T) {
	t.Parallel()

	stretched := 131_072
	provider := &Provider{Model: "qwen2.5-coder:32b", ContextWindowTokens: &stretched}
	assert.Equal(t, 131_072, provider.ContextWindow())
	assert.Equal(t, 131_072, provider.ConfiguredContextWindow())

	provider.ContextWindowTokens = nil
	assert.Equal(t, 32_768, provider.ContextWindow(), "left unset, the id is read")
	assert.Zero(t, provider.ConfiguredContextWindow())

	assert.Equal(t, 16_384, aimodel.WindowOr(16_384, "llama3.3:70b"))
	assert.Equal(t, 128_000, aimodel.WindowOr(0, "llama3.3:70b"))
}

/*
Only a Claude model behind the Anthropic protocol is sent files as themselves:
the same protocol fronts other vendors' models that refuse a document block,
and the other protocols are sent the text read from each file.
*/
func TestFileInput_OnlyAClaudeModelReadsFilesItself(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		kind  Kind
		model string
		reads bool
		pages int
	}{
		{kind: KindAnthropicMessages, model: "claude-opus-5-5", reads: true, pages: 600},
		{kind: KindAnthropicMessages, model: "anthropic.claude-sonnet-4-6", reads: true, pages: 600},
		{kind: KindAnthropicMessages, model: "claude-haiku-4-5-20251001", reads: true, pages: 100},
		{kind: KindAnthropicMessages, model: "kimi-k2"},
		{kind: KindAnthropicMessages, model: "claude-latest"},
		{kind: KindOpenAIResponses, model: "claude-opus-5-5"},
		{kind: KindOpenAIChat, model: "Qwen/Qwen2.5-VL-7B-Instruct"},
	} {
		provider := &Provider{Kind: tc.kind, Model: tc.model}
		input, reads := provider.FileInput()
		assert.Equal(t, tc.reads, reads, "%s %s", tc.kind, tc.model)
		assert.Equal(t, tc.pages, input.MaxPDFPages, "%s %s", tc.kind, tc.model)
	}
}
