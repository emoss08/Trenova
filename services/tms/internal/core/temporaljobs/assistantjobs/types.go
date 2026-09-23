// Package assistantjobs answers an assistant turn as a workflow.
//
// A chat turn used to occupy a Gin goroutine for its whole life, which made
// the reply a property of the connection carrying it: a deploy ended every
// conversation in flight. The turn is now a workflow of its own, answered in
// three steps:
//
//  1. Prepare reads what the question needs, checks it may be answered and
//     runs the scope guard.
//  2. The agent loop runs in workflow code (agentflow), one activity per
//     model call and per tool call, so a failure retries the step that
//     failed and a lost worker's turn resumes where it was.
//  3. Finish saves what the turn came to and closes its record.
//
// The reader follows the turn on a Workflow Stream the workflow hosts. The
// stream exists as soon as the workflow does, which is before the request
// that started it returns, so a reader can never arrive ahead of it.
package assistantjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const AssistantTurnWorkflowName = "AssistantTurnWorkflow"

// WorkflowIDFor names the execution carrying a turn.
func WorkflowIDFor(turnID pulid.ID) string {
	return conversation.AssistantTurnWorkflowID(turnID)
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

func (p *AssistantTurnPayload) sendRequest() *serviceports.SendMessageRequest {
	return &serviceports.SendMessageRequest{
		ThreadID:              p.ThreadID,
		Content:               p.Content,
		Page:                  p.Request.Page,
		TenantInfo:            p.tenantInfo(),
		PreferredProviderID:   p.Request.PreferredProviderID,
		ProviderChosen:        p.Request.ProviderChosen,
		AttachmentDocumentIDs: p.Request.AttachmentDocumentIDs,
		Mentions:              p.Request.Mentions,
		FollowUpProposalID:    p.Request.FollowUpProposalID,
		FollowUpPlanID:        p.Request.FollowUpPlanID,
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
	FollowUpPlanID        pulid.ID           `json:"followUpPlanId,omitempty"`
	// Awaited says the request that asked is waiting for the turn's result
	// rather than reading its stream. That caller has the reply as soon as
	// the turn ends, so nobody is told later that it is ready.
	Awaited bool `json:"awaited,omitempty"`
}

// AssistantTurnResult is what the turn came to.
type AssistantTurnResult struct {
	Status  string `json:"status"`
	Refused bool   `json:"refused"`
	// Result is the saved turn, when it finished. A turn that failed or was
	// stopped has Message instead, written for the person who asked.
	Result  *serviceports.SendMessageResult `json:"result,omitempty"`
	Message string                          `json:"message,omitempty"`
}

// FinishTurnInput is everything the turn did, for saving.
type FinishTurnInput struct {
	Payload *AssistantTurnPayload `json:"payload"`
	// Plan is nil when the question was turned away before it was planned:
	// then there is nothing to save, only the turn's record to close.
	Plan *assistantservice.TurnPlan `json:"plan,omitempty"`
	// Rejection is why the question was turned away, written for the person
	// who asked.
	Rejection string                        `json:"rejection,omitempty"`
	Run       *serviceports.RunResult       `json:"run,omitempty"`
	Failure   *modelcall.Failure            `json:"failure,omitempty"`
	Artifacts []*assistantartifact.Artifact `json:"artifacts,omitempty"`
	Events    []temporaltype.StreamItem     `json:"events,omitempty"`
}

// TurnEnding is how the turn ended: its result, and the last event its reader
// is sent.
type TurnEnding struct {
	Result AssistantTurnResult     `json:"result"`
	Event  temporaltype.StreamItem `json:"event"`
}

// NotifyUnseenTurnInput is a turn that ended with nobody reading it, for
// telling the person who asked.
type NotifyUnseenTurnInput struct {
	Payload *AssistantTurnPayload `json:"payload"`
	// Status is how the turn ended.
	Status conversation.AssistantTurnStatus `json:"status"`
}
