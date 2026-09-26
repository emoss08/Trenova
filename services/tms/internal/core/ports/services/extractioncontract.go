package services

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

type ExtractionContract interface {
	CompletionRequest(fileName string, pages []AIDocumentPage) *StructuredCompletionRequest
	PageLimit() int
	FieldKeys() []string
	ParseReply(text string) (*AIExtractResult, error)
}

type RenderedPrompt struct {
	System      string   `json:"system"`
	User        string   `json:"user"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"topP,omitempty"`
}

type StructuredPromptRenderer interface {
	RenderStructuredPrompt(
		req *StructuredCompletionRequest,
		mode aiprovider.StructuredOutputMode,
	) RenderedPrompt
}

type ExtractionReplyReader interface {
	ReadReply(text string) (*aicorrection.Prediction, error)
}
