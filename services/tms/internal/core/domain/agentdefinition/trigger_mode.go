package agentdefinition

import "github.com/emoss08/trenova/internal/core/domain/agent"

type TriggerMode string

const (
	TriggerChat       = TriggerMode("Chat")
	TriggerScheduled  = TriggerMode("Scheduled")
	TriggerEvent      = TriggerMode("Event")
	TriggerContinuous = TriggerMode("Continuous")
)

func (m TriggerMode) IsValid() bool {
	switch m {
	case TriggerChat, TriggerScheduled, TriggerEvent, TriggerContinuous:
		return true
	default:
		return false
	}
}

func (m TriggerMode) RunTrigger() agent.RunTrigger {
	switch m {
	case TriggerScheduled:
		return agent.RunTriggerScheduled
	case TriggerEvent:
		return agent.RunTriggerEvent
	case TriggerContinuous:
		return agent.RunTriggerContinuous
	default:
		return agent.RunTriggerChat
	}
}

type OutputMode string

const (
	OutputConversational = OutputMode("Conversational")
	OutputReport         = OutputMode("Report")
)

func (m OutputMode) IsValid() bool {
	switch m {
	case OutputConversational, OutputReport:
		return true
	default:
		return false
	}
}

type ContextProvider string

const (
	ContextOrganization = ContextProvider("Organization")
	ContextClock        = ContextProvider("Clock")
	ContextUser         = ContextProvider("User")
	ContextPage         = ContextProvider("Page")
	ContextTools        = ContextProvider("Tools")
	ContextMemory       = ContextProvider("Memory")
)

func (p ContextProvider) IsValid() bool {
	switch p {
	case ContextOrganization, ContextClock, ContextUser, ContextPage, ContextTools, ContextMemory:
		return true
	default:
		return false
	}
}

func AllContextProviders() []ContextProvider {
	return []ContextProvider{
		ContextOrganization,
		ContextClock,
		ContextUser,
		ContextPage,
		ContextTools,
		ContextMemory,
	}
}
