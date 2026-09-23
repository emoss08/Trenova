// Package assistantjobs runs an interactive assistant turn on a worker
// instead of on the request that asked for it.
//
// A chat turn used to occupy a Gin goroutine for its whole life, which made
// the reply a property of the connection carrying it: a deploy ended every
// conversation in flight, and the server's write deadline had to be lifted
// by hand because a reasoning model plus two tool calls outlives it. Moving
// the work here leaves the API holding nothing but a relay.
//
// The turn's events reach the reader through its redis stream rather than
// through this workflow's history. Workflow history is a durable record of
// decisions, not a pipe for sixty tokens a second, and a reader rejoining a
// reply needs a cursor into the text — which is what the stream's entry ids
// already are.
package assistantjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	AssistantTurnWorkflowName = "AssistantTurnWorkflow"

	// workflowIDPrefix makes a turn's execution findable from its record, and
	// its id unique: one turn, one execution, for ever.
	workflowIDPrefix = "assistant-turn:"
)

// WorkflowIDFor names the execution carrying a turn.
func WorkflowIDFor(turnID pulid.ID) string {
	return workflowIDPrefix + turnID.String()
}

// AssistantTurnPayload is one question and everything needed to answer it.
//
// The actor travels with the turn rather than being rebuilt on the worker.
// A turn runs as the person who asked and as nobody else, and a worker that
// reconstructed an actor from a tenant would be inventing an authority that
// nobody granted.
type AssistantTurnPayload struct {
	temporaltype.BasePayload

	TurnID   pulid.ID                  `json:"turnId"`
	ThreadID pulid.ID                  `json:"threadId"`
	Actor    serviceports.RequestActor `json:"actor"`
	Content  string                    `json:"content"`
	Request  AssistantTurnRequest      `json:"request"`
}

func (p *AssistantTurnPayload) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.Actor.UserID,
	}
}

// AssistantTurnRequest is what the person handed over with the message.
type AssistantTurnRequest struct {
	Page                  *agent.PageContext `json:"page,omitempty"`
	Mentions              []agent.EntityRef  `json:"mentions,omitempty"`
	AttachmentDocumentIDs []pulid.ID         `json:"attachmentDocumentIds,omitempty"`
	PreferredProviderID   pulid.ID           `json:"preferredProviderId,omitempty"`
	ProviderChosen        bool               `json:"providerChosen"`
	FollowUpProposalID    pulid.ID           `json:"followUpProposalId,omitempty"`
}

// AssistantTurnResult is what the turn came to.
type AssistantTurnResult struct {
	Status  string `json:"status"`
	Refused bool   `json:"refused"`
	// Replayed says a later attempt found the turn already answered and did
	// not answer it again.
	Replayed bool `json:"replayed"`
}
