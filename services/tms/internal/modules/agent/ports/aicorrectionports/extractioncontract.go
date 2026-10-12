package aicorrectionports

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
)

type ExtractionContract interface {
	CompletionRequest(
		fileName string,
		pages []documentaiextractionports.AIDocumentPage,
	) *agentports.StructuredCompletionRequest
	PageLimit() int
	FieldKeys() []string
	ParseReply(text string) (*documentaiextractionports.AIExtractResult, error)
}

type RenderedPrompt struct {
	System      string   `json:"system"`
	User        string   `json:"user"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"topP,omitempty"`
}

type StructuredPromptRenderer interface {
	RenderStructuredPrompt(
		req *agentports.StructuredCompletionRequest,
		mode aiprovider.StructuredOutputMode,
	) RenderedPrompt
}

type ExtractionReplyReader interface {
	ReadReply(text string) (*aicorrection.Prediction, error)
}
