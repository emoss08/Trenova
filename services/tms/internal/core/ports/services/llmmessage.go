package services

// Role identifies who produced a message in a conversation.
type Role string

const (
	RoleUser      = Role("user")
	RoleAssistant = Role("assistant")
	// RoleTool carries the result of a tool the assistant asked for. Every
	// protocol represents this differently, which is the adapters' problem rather
	// than the caller's.
	RoleTool = Role("tool")
)

// Message is one turn. A single assistant turn can both say something and ask
// for tools, so Content and ToolCalls are not mutually exclusive.
type Message struct {
	Role    Role
	Content string
	// ToolCalls is set on an assistant turn that asked for tools.
	ToolCalls []ToolCall
	// ToolCallID and ToolName are set on a RoleTool turn, tying the result back to
	// the request that produced it.
	ToolCallID string
	ToolName   string
	// IsError marks a tool result that failed, so the model can recover rather
	// than treating the message as data.
	IsError bool
}

// ToolSpec describes a tool to the model.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall is the model asking for a tool.
type ToolCall struct {
	// ID ties a call to its result. Protocols that do not supply one get a
	// synthesized ID so the loop can still pair them up.
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	// ArgumentsError is set when the provider's argument text did not parse —
	// usually because the reply hit its output limit partway through the JSON.
	// The runtime refuses such a call rather than running the tool on the
	// empty map that used to stand in for it.
	ArgumentsError string `json:"argumentsError,omitempty"`
}

// UserMessage is the common single-turn case.
func UserMessage(content string) []Message {
	return []Message{{Role: RoleUser, Content: content}}
}
