package conversation

const (
	ProviderBlockServerToolUse     = "server_tool_use"
	ProviderBlockAdvisorToolResult = "advisor_tool_result"
)

type ProviderBlock struct {
	Type      string            `json:"type"`
	Text      string            `json:"text,omitempty"`
	Signature string            `json:"signature,omitempty"`
	Data      string            `json:"data,omitempty"`
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name,omitempty"`
	Input     map[string]any    `json:"input,omitempty"`
	ToolUseID string            `json:"toolUseId,omitempty"`
	Result    *ServerToolResult `json:"result,omitempty"`
	Model     string            `json:"model,omitempty"`
}

type ServerToolResult struct {
	Type             string `json:"type"`
	Text             string `json:"text,omitempty"`
	EncryptedContent string `json:"encryptedContent,omitempty"`
	StopReason       string `json:"stopReason,omitempty"`
	ErrorCode        string `json:"errorCode,omitempty"`
}

func (t *ReasoningTrace) AdvisorModel() string {
	if t == nil {
		return ""
	}
	for idx := len(t.Blocks) - 1; idx >= 0; idx-- {
		block := &t.Blocks[idx]
		if block.Type == ProviderBlockAdvisorToolResult && block.Model != "" {
			return block.Model
		}
	}

	return ""
}

func (t *ReasoningTrace) WithoutServerTools() *ReasoningTrace {
	if t == nil {
		return nil
	}
	kept := make([]ProviderBlock, 0, len(t.Blocks))
	for idx := range t.Blocks {
		switch t.Blocks[idx].Type {
		case ProviderBlockServerToolUse, ProviderBlockAdvisorToolResult:
			continue
		}
		kept = append(kept, t.Blocks[idx])
	}
	if len(kept) == len(t.Blocks) {
		return t
	}
	trace := *t
	trace.Blocks = kept

	return &trace
}
