package modeladapter

const (
	ThinkingAnswerRoom   = 2048
	thinkingFloor        = 8192
	ReasoningAnswerFloor = thinkingFloor + ThinkingAnswerRoom
)

func (c *Call) Reasons() bool {
	if c.ReasoningEffort().Enabled() {
		return true
	}
	if c.Provider == nil || c.Request == nil {
		return false
	}

	kind := string(c.Provider.Kind)
	for idx := range c.Request.Messages {
		message := &c.Request.Messages[idx]
		if message.Role != RoleAssistant || message.Reasoning == nil {
			continue
		}
		if message.Reasoning.ProviderKind != kind {
			continue
		}
		if message.Reasoning.Text != "" || message.Reasoning.Encrypted != "" {
			return true
		}
	}

	return false
}
