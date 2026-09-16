package modeladapter

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

const anthropicVersion = "2023-06-01"

type anthropicAdapter struct{}

// NewAnthropicAdapter speaks the Anthropic Messages API. Bedrock's mantle
// endpoint serves this same shape, so pointing a provider's base URL there needs
// no separate adapter.
func NewAnthropicAdapter() Adapter { return anthropicAdapter{} }

func (anthropicAdapter) Kind() aiprovider.Kind { return aiprovider.KindAnthropicMessages }

type anthropicRequest struct {
	Model        string                 `json:"model"`
	MaxTokens    int                    `json:"max_tokens"`
	System       string                 `json:"system,omitempty"`
	Messages     []anthropicMessage     `json:"messages"`
	OutputConfig *anthropicOutputConfig `json:"output_config,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicOutputConfig struct {
	Format anthropicOutputFormat `json:"format"`
}

type anthropicOutputFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

type anthropicResponse struct {
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason"`
	Content    []anthropicContentBlock `json:"content"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (a anthropicAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := anthropicRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		System:    call.Request.System,
		Messages: []anthropicMessage{
			{Role: "user", Content: call.Request.UserContent},
		},
	}

	if schema := call.Request.OutputSchema; schema != nil &&
		call.Provider.StructuredOutputMode == aiprovider.StructuredOutputJSONSchema {
		body.OutputConfig = &anthropicOutputConfig{
			Format: anthropicOutputFormat{Type: "json_schema", Schema: schema},
		}
	}

	var envelope anthropicResponse
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/v1/messages",
		map[string]string{
			"x-api-key":         call.APIKey,
			"anthropic-version": anthropicVersion,
		},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	return &Response{
		Text:            firstAnthropicText(&envelope),
		ModelIdentifier: envelope.Model,
		InputTokens:     envelope.Usage.InputTokens,
		OutputTokens:    envelope.Usage.OutputTokens,
		Refused:         envelope.StopReason == "refusal",
	}, nil
}

func firstAnthropicText(resp *anthropicResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return block.Text
		}
	}

	return ""
}
