package agentports

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
)

// Role identifies who produced a message in a conversation.
type Role string

const RoleUser = Role("user")

const RoleAssistant = Role("assistant")

// RoleTool carries the result of a tool the assistant asked for. Every
// protocol represents this differently, which is the adapters' problem rather
// than the caller's.
const RoleTool = Role("tool")

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
	// Reasoning is the thinking behind an assistant turn, replayed so a
	// provider that signs or references its reasoning can continue it.
	Reasoning *conversation.ReasoningTrace
	// Files are the files the person attached to this user message. A model
	// that reads files is sent each it can take as the file itself, in place
	// of its entry in Content; any other reads Content as it is.
	Files []MessageFile
}

// MessageFile is a file attached to a user message, named by its document.
// Only the reference crosses a durable boundary: the bytes are read where the
// model is called.
type MessageFile struct {
	DocumentID  string
	FileName    string
	ContentType string
	Size        int64
	// PageCount is what reading the file counted, zero when it is not known.
	PageCount int
	// EntryStart and EntryEnd bound the file's entry in the message's
	// Content, by byte, and NativeEntry is what replaces it when the file
	// itself is sent. Offsets rather than a copy, because the entry holds the
	// text read from the file and the message crosses every model call.
	EntryStart  int
	EntryEnd    int
	NativeEntry string
}

// ToolCall is the model asking for a tool.
type ToolCall struct {
	// ID ties a call to its result. Protocols that do not supply one get a
	// synthesized ID so the loop can still pair them up.
	ID string `json:"id"`
	// SynthesizedID marks an ID the adapter made up from the call's position
	// because the protocol gave none. It is unique within one response and
	// nowhere else, so the loop replaces it with an ID of its own.
	SynthesizedID bool           `json:"synthesizedId,omitempty"`
	Name          string         `json:"name"`
	Arguments     map[string]any `json:"arguments"`
	// ArgumentsError is set when the provider's argument text did not parse —
	// usually because the reply hit its output limit partway through the JSON.
	// The runtime refuses such a call rather than running the tool on the
	// empty map that used to stand in for it.
	ArgumentsError string `json:"argumentsError,omitempty"`
	// ProviderData is what the provider attached to the call besides its
	// name and arguments and wants back with it. Gemini signs each function
	// call with a thought signature that the OpenAI-compatible protocol
	// carries under extra_content, and it rejects a follow-up request that
	// replays the call without it. The data is opaque and is replayed only
	// to the provider that produced it, named by ProviderID.
	ProviderData map[string]any `json:"providerData,omitempty"`
	ProviderID   pulid.ID       `json:"providerId,omitempty"`
	// Why is the step's rationale, lifted out of the arguments by the
	// runtime; see agentruntime/rationale.go.
	Why *conversation.StepRationale `json:"why,omitempty"`
}

// UserMessage is the common single-turn case.
func UserMessage(content string) []Message {
	return []Message{{Role: RoleUser, Content: content}}
}
