package runtimetools

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
)

// Turn is what the runtime's own tools read of the turn that calls them:
// what it was asked and shown, what it has filed and what it has asked the
// person to decide.
type Turn interface {
	Request() *agentruntimeports.RunRequest
	Messages() []llmports.Message
	Result() *agentruntimeports.RunResult
	Decisions() map[pulid.ID]struct{}
}
