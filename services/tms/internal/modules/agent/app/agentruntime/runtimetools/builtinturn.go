package runtimetools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/shared/pulid"
)

// Turn is what the runtime's own tools read of the turn that calls them:
// what it was asked and shown, what it has filed and what it has asked the
// person to decide.
type Turn interface {
	replycheck.Turn
	Decisions() map[pulid.ID]struct{}
}
