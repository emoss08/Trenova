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
