package anthropiccompletionservice

import "github.com/emoss08/trenova/internal/core/domain/agent"

const (
	anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion     = "2023-06-01"
	defaultModel         = "claude-opus-4-8"
	defaultMaxTokens     = 8192
)

type messagesRequest struct {
	Model        string        `json:"model"`
	MaxTokens    int           `json:"max_tokens"`
	System       string        `json:"system,omitempty"`
	Messages     []message     `json:"messages"`
	OutputConfig *outputConfig `json:"output_config,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type outputConfig struct {
	Format outputFormat `json:"format"`
}

type outputFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

type messagesResponse struct {
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Content    []contentBlock `json:"content"`
	Usage      usage          `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicErrorEnvelope struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type diagnosisPayload struct {
	Proposals  []proposalPayload  `json:"proposals"`
	Exceptions []exceptionPayload `json:"exceptions"`
}

type proposalPayload struct {
	ToolName   string              `json:"toolName"`
	ToolParams map[string]any      `json:"toolParams"`
	Confidence float64             `json:"confidence"`
	Rationale  string              `json:"rationale"`
	Evidence   []agent.EvidenceRef `json:"evidence"`
}

type exceptionPayload struct {
	Category       string              `json:"category"`
	Severity       string              `json:"severity"`
	AttemptSummary string              `json:"attemptSummary"`
	Evidence       []agent.EvidenceRef `json:"evidence"`
	BlastRadius    int                 `json:"blastRadius"`
}
