package aiprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestFileInput_SendsPDFsAsDocumentsAndFourImageFormatsAsImages(t *testing.T) {
	t.Parallel()

	var input FileInput
	assert.Equal(t, FileKindDocument, input.Kind("application/pdf"))
	assert.Equal(t, FileKindImage, input.Kind("Image/PNG; charset=binary"))
	assert.Equal(t, FileKindImage, input.Kind("image/jpeg"))
	assert.Equal(t, FileKindImage, input.Kind("image/gif"))
	assert.Equal(t, FileKindImage, input.Kind("image/webp"))
	assert.Equal(t, FileKindNone, input.Kind("image/tiff"))
	assert.Equal(t, FileKindNone, input.Kind("image/heic"))
	assert.Equal(t, FileKindNone, input.Kind("text/csv"))
	assert.Equal(t, FileKindNone, input.Kind(""))
	assert.Equal(t, "image/png", MediaType(" Image/PNG; charset=binary"))
}
