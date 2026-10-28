package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// HandoffThreadRequest takes a person's conversation to another agent.
type HandoffThreadRequest struct {
	TenantInfo        pagination.TenantInfo
	ThreadID          pulid.ID
	AgentDefinitionID pulid.ID
	// Facts are pinned facts the person adds for the hand-off, carried
	// after the conversation's own.
	Facts []string
	// ChapterID hands off one chapter: the new conversation is named for it
	// and briefed from its messages alone. With a chapter the target may be
	// the conversation's own agent, which continues the chapter on its own.
	ChapterID pulid.ID
}

// HandoffThreadResult is the conversation the hand-off started and the card
// it left in the one handed off.
type HandoffThreadResult struct {
	Thread  *conversation.Thread  `json:"thread"`
	Message *conversation.Message `json:"message"`
}

type AssistantHandoffService interface {
	Handoff(
		ctx context.Context,
		req *HandoffThreadRequest,
		actor *permissionports.RequestActor,
	) (*HandoffThreadResult, error)
}
