package agent

import "github.com/emoss08/trenova/shared/pulid"

const (
	TranscriptMessageBytes = 64 * 1024
	TranscriptBytes        = 256 * 1024
)

type RunTranscript struct {
	Messages        []TranscriptMessage `json:"messages"`
	OmittedMessages int                 `json:"omittedMessages,omitempty"`
	OmittedAt       int                 `json:"omittedAt,omitempty"`
}

type TranscriptMessage struct {
	Role              string               `json:"role"`
	Kind              string               `json:"kind,omitempty"`
	Content           string               `json:"content,omitempty"`
	Reasoning         string               `json:"reasoning,omitempty"`
	ToolCalls         []TranscriptToolCall `json:"toolCalls,omitempty"`
	ToolCallID        string               `json:"toolCallId,omitempty"`
	ToolName          string               `json:"toolName,omitempty"`
	ToolFailed        bool                 `json:"toolFailed,omitempty"`
	ToolVerdict       string               `json:"toolVerdict,omitempty"`
	ToolSummary       string               `json:"toolSummary,omitempty"`
	AgentDefinitionID pulid.ID             `json:"agentId,omitempty"`
	DelegateCallID    string               `json:"delegateCallId,omitempty"`
	Omitted           bool                 `json:"omitted,omitempty"`
	CreatedAt         int64                `json:"createdAt"`
}

type TranscriptToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

func (m *TranscriptMessage) WithoutBody() TranscriptMessage {
	bare := *m
	bare.Content = ""
	bare.Reasoning = ""
	if len(m.ToolCalls) > 0 {
		bare.ToolCalls = make([]TranscriptToolCall, len(m.ToolCalls))
		for idx, call := range m.ToolCalls {
			bare.ToolCalls[idx] = TranscriptToolCall{ID: call.ID, Name: call.Name}
		}
	}
	bare.Omitted = true

	return bare
}
