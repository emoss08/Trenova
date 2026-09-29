package modeladapter

const (
	thinkingAnswerRoom   = 2048
	thinkingFloor        = 8192
	reasoningAnswerFloor = thinkingFloor + thinkingAnswerRoom
)

func answerRoom(call *Call, ceiling int) int {
	if !call.reasons() {
		return ceiling
	}

	return max(ceiling, reasoningAnswerFloor)
}

func (c *Call) reasons() bool {
	if c.reasoning().Enabled() {
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
