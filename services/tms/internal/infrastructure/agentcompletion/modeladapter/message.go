package modeladapter

import (
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// The conversation primitives live in the ports package because the port
// interfaces are expressed in terms of them, and core must not import
// infrastructure. They are aliased here so adapter code reads in its own terms
// rather than qualifying every message type.
type (
	Role     = agentports.Role
	Message  = agentports.Message
	ToolSpec = kernelports.ToolSpec
	ToolCall = agentports.ToolCall
	// ReasoningTrace is the conversation domain's, so what an adapter reads
	// off the wire is what the message stores and what the next call replays.
	ReasoningTrace = conversation.ReasoningTrace
	CutOffToolCall = agentports.CutOffToolCall
)

const (
	RoleUser      = agentports.RoleUser
	RoleAssistant = agentports.RoleAssistant
	RoleTool      = agentports.RoleTool
)

// UserMessage is the common single-turn case.
func UserMessage(content string) []Message {
	return agentports.UserMessage(content)
}
