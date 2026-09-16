package modeladapter

import serviceports "github.com/emoss08/trenova/internal/core/ports/services"

// The conversation primitives live in the ports package because the port
// interfaces are expressed in terms of them, and core must not import
// infrastructure. They are aliased here so adapter code reads in its own terms
// rather than qualifying every message type.
type (
	Role     = serviceports.Role
	Message  = serviceports.Message
	ToolSpec = serviceports.ToolSpec
	ToolCall = serviceports.ToolCall
)

const (
	RoleUser      = serviceports.RoleUser
	RoleAssistant = serviceports.RoleAssistant
	RoleTool      = serviceports.RoleTool
)

// UserMessage is the common single-turn case.
func UserMessage(content string) []Message {
	return serviceports.UserMessage(content)
}
