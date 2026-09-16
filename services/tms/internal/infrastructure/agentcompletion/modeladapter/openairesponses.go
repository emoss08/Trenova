package modeladapter

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

type openAIResponsesAdapter struct{}

// NewOpenAIResponsesAdapter speaks the OpenAI Responses API, which the document
// intelligence path already uses and which Bedrock's mantle endpoint also serves.
func NewOpenAIResponsesAdapter() Adapter { return openAIResponsesAdapter{} }

func (openAIResponsesAdapter) Kind() aiprovider.Kind { return aiprovider.KindOpenAIResponses }

type responsesRequest struct {
	Model           string               `json:"model"`
	Input           []responsesMessage   `json:"input"`
	Text            *responsesTextConfig `json:"text,omitempty"`
	MaxOutputTokens int                  `json:"max_output_tokens,omitempty"`
}

type responsesMessage struct {
	Role    string                 `json:"role"`
	Content []responsesMessagePart `json:"content"`
}

type responsesMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesTextConfig struct {
	Format responsesFormat `json:"format"`
}

type responsesFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name,omitempty"`
	Schema map[string]any `json:"schema,omitempty"`
	Strict bool           `json:"strict,omitempty"`
}

type responsesEnvelope struct {
	Model  string          `json:"model"`
	Status string          `json:"status"`
	Output []responsesItem `json:"output"`
	Usage  responsesUsage  `json:"usage"`
}

type responsesItem struct {
	Type    string              `json:"type"`
	Content []responsesItemPart `json:"content"`
}

type responsesItemPart struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (a openAIResponsesAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := responsesRequest{
		Model:           call.Provider.Model,
		MaxOutputTokens: call.Request.MaxTokens,
		Input: []responsesMessage{
			{
				Role:    "system",
				Content: []responsesMessagePart{{Type: "input_text", Text: call.Request.System}},
			},
			{
				Role:    "user",
				Content: []responsesMessagePart{{Type: "input_text", Text: call.Request.UserContent}},
			},
		},
	}

	if schema := call.Request.OutputSchema; schema != nil &&
		call.Provider.StructuredOutputMode == aiprovider.StructuredOutputJSONSchema {
		body.Text = &responsesTextConfig{
			Format: responsesFormat{
				Type:   "json_schema",
				Name:   call.Request.SchemaName,
				Schema: schema,
				Strict: true,
			},
		}
	}

	var envelope responsesEnvelope
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/responses",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	text, refused := firstResponsesText(&envelope)

	return &Response{
		Text:            text,
		ModelIdentifier: firstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.Usage.InputTokens,
		OutputTokens:    envelope.Usage.OutputTokens,
		Refused:         refused,
	}, nil
}

func firstResponsesText(resp *responsesEnvelope) (string, bool) {
	for idx := range resp.Output {
		item := &resp.Output[idx]
		for partIdx := range item.Content {
			part := &item.Content[partIdx]
			if strings.TrimSpace(part.Refusal) != "" {
				return "", true
			}
			if strings.TrimSpace(part.Text) != "" {
				return part.Text, false
			}
		}
	}

	return "", false
}
