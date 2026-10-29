package replycheck

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// Turn is what the checks of a reply read of the turn that wrote it:
// what it was asked, shown and read, what it filed and what it put beside
// the conversation.
type Turn interface {
	Request() *agentruntimeports.RunRequest
	Messages() []llmports.Message
	System() string
	Evidence() []string
	Result() *agentruntimeports.RunResult
	Artifacts() []agentruntimeports.ShownArtifact
	Holds(name string) bool
}

// SpecFor is the spec of a tool by its name, false for a tool nobody
// registered.
type SpecFor func(name string) (kernelports.ToolSpec, bool)
