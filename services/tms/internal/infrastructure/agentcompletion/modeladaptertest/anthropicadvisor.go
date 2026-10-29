package modeladaptertest

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

func AdvisorRequest() modeladapter.Request {
	return modeladapter.Request{
		System:   "Rules.",
		Messages: modeladapter.UserMessage("Where is S1?"),
		Tools:    WireTools(),
		Advisor:  llmports.DefaultAdvisor(""),
	}
}
