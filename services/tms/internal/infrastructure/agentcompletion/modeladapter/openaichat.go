package modeladapter

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

type openAIChatAdapter struct{}

// NewOpenAIChatAdapter speaks the OpenAI Chat Completions API, which is the de
// facto interchange format: vLLM, SGLang, LM Studio, llama.cpp, OpenRouter, Groq,
// Together, Fireworks, DeepInfra and Bedrock's chat-completions endpoint all
// accept it. What varies between them is not the request shape but how much of
// response_format they honour, which is why the provider declares its own
// structured-output mode instead of the adapter assuming one.
func NewOpenAIChatAdapter() Adapter { return openAIChatAdapter{} }

func (openAIChatAdapter) Kind() aiprovider.Kind { return aiprovider.KindOpenAIChat }

type chatRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	MaxTokens      int                 `json:"max_completion_tokens,omitempty"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
	Stream         bool                `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *chatJSONSchema `json:"json_schema,omitempty"`
}

type chatJSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type chatResponse struct {
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
}

type chatChoice struct {
	FinishReason string      `json:"finish_reason"`
	Message      chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func (a openAIChatAdapter) Complete(ctx context.Context, call *Call) (*Response, error) {
	body := chatRequest{
		Model:     call.Provider.Model,
		MaxTokens: call.Request.MaxTokens,
		Messages: []chatMessage{
			{Role: "system", Content: call.Request.System},
			{Role: "user", Content: call.Request.UserContent},
		},
		Stream: false,
	}
	body.ResponseFormat = chatResponseFormatFor(call)

	var envelope chatResponse
	err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+"/chat/completions",
		map[string]string{"Authorization": bearer(call.APIKey)},
		body,
		&envelope,
	)
	if err != nil {
		return nil, err
	}

	text, refused := firstChatText(&envelope)

	return &Response{
		Text:            text,
		ModelIdentifier: firstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.Usage.PromptTokens,
		OutputTokens:    envelope.Usage.CompletionTokens,
		Refused:         refused,
	}, nil
}

// chatResponseFormatFor asks for exactly as much enforcement as the endpoint was
// declared to support. Sending a json_schema to a server that ignores it is worse
// than not sending one, because the reply comes back unconstrained with no error
// to signal that the schema was dropped.
func chatResponseFormatFor(call *Call) *chatResponseFormat {
	if call.Request.OutputSchema == nil {
		return nil
	}

	switch call.Provider.StructuredOutputMode {
	case aiprovider.StructuredOutputJSONSchema:
		return &chatResponseFormat{
			Type: "json_schema",
			JSONSchema: &chatJSONSchema{
				Name:   call.Request.SchemaName,
				Schema: call.Request.OutputSchema,
				Strict: true,
			},
		}
	case aiprovider.StructuredOutputJSONMode:
		return &chatResponseFormat{Type: "json_object"}
	case aiprovider.StructuredOutputPrompted:
		// The schema rides in the system prompt instead; the reply is repaired
		// and validated on receipt.
		return nil
	default:
		return nil
	}
}

func firstChatText(resp *chatResponse) (string, bool) {
	for idx := range resp.Choices {
		choice := &resp.Choices[idx]
		if choice.FinishReason == "content_filter" {
			return "", true
		}
		if text := strings.TrimSpace(choice.Message.Content); text != "" {
			return choice.Message.Content, false
		}
	}

	return "", false
}

func bearer(apiKey string) string {
	if strings.TrimSpace(apiKey) == "" {
		return ""
	}

	return "Bearer " + apiKey
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
