package assistantcontract

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const CompactionWorkflowName = "ConversationCompactionWorkflow"

// CompactionPayload is one compaction of a conversation.
type CompactionPayload struct {
	temporaltype.BasePayload

	TurnID   pulid.ID                     `json:"turnId"`
	ThreadID pulid.ID                     `json:"threadId"`
	Actor    permissionports.RequestActor `json:"actor"`
	// Auto says the conversation is compacting itself.
	Auto bool `json:"auto"`
}

func (p *CompactionPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.Actor.UserID,
	}
}

// StartCompactionWorkflow hands a recorded compaction turn to a worker.
func StartCompactionWorkflow(
	ctx context.Context,
	workflows kernelports.WorkflowStarter,
	turn *conversation.AssistantTurn,
	actor *permissionports.RequestActor,
	auto bool,
) (client.WorkflowRun, error) {
	return workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       WorkflowIDFor(turn.ID),
		TaskQueue:                                temporaltype.TaskQueueAgentChat.String(),
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary:                            "Compact a conversation",
		Priority: temporal.Priority{
			PriorityKey: agentflowcontract.PriorityInteractive,
			FairnessKey: actor.OrganizationID.String(),
		},
	}, CompactionWorkflowName, &CompactionPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			UserID:         actor.UserID,
			Timestamp:      timeutils.NowUnix(),
		},
		TurnID:   turn.ID,
		ThreadID: turn.ThreadID,
		Actor:    *actor,
		Auto:     auto,
	})
}

// StartCompactionRequest asks for a conversation to be compacted.
type StartCompactionRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Actor      permissionports.RequestActor
	Auto       bool
}

// CompactionTurns is the part of the turn service starting a compaction
// needs.
type CompactionTurns interface {
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
}

// StartCompaction records a compaction as the conversation's live turn and
// hands it to a worker. A person asking and a conversation compacting itself
// both start here.
func StartCompaction(
	ctx context.Context,
	turns CompactionTurns,
	workflows kernelports.WorkflowStarter,
	req StartCompactionRequest,
) (*conversation.AssistantTurn, error) {
	if workflows == nil {
		return nil, errors.New("no workflow client to start a compaction with")
	}

	return turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID:   req.ThreadID,
		UserID:     req.Actor.UserID,
		TenantInfo: req.TenantInfo,
		Origin:     conversation.AssistantTurnOriginCompaction,
	}, func(turn *conversation.AssistantTurn) (string, error) {
		run, err := StartCompactionWorkflow(ctx, workflows, turn, &req.Actor, req.Auto)
		if err != nil {
			return "", err
		}

		return run.GetID(), nil
	})
}

const AgentDryRunWorkflowName = "AgentDryRunWorkflow"

// DryRunTimeout bounds a dry run; a person is watching it.
const DryRunTimeout = 5 * time.Minute

// DryRunWorkflowIDFor names the execution carrying a dry run.
func DryRunWorkflowIDFor(runID pulid.ID) string {
	return "agent-dry-run:" + runID.String()
}

// AgentDryRunPayload is a draft agent and the message it is tried with. The
// draft travels whole: it is not saved, so there is nothing to read it back
// from.
type AgentDryRunPayload struct {
	temporaltype.BasePayload

	RunID      pulid.ID                     `json:"runId"`
	Actor      permissionports.RequestActor `json:"actor"`
	Definition *agentdefinition.Definition  `json:"definition"`
	Prompt     string                       `json:"prompt"`
}

// StartDryRunWorkflow starts a dry run on the chat queue, where somebody is
// watching it arrive.
func StartDryRunWorkflow(
	ctx context.Context,
	workflows kernelports.WorkflowStarter,
	payload *AgentDryRunPayload,
) (client.WorkflowRun, error) {
	return workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       DryRunWorkflowIDFor(payload.RunID),
		TaskQueue:                                temporaltype.TaskQueueAgentChat.String(),
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		WorkflowExecutionTimeout:                 DryRunTimeout + agentflowcontract.DrainWindow,
		StaticSummary:                            "Agent dry run",
		Priority: temporal.Priority{
			PriorityKey: agentflowcontract.PriorityInteractive,
			FairnessKey: payload.Actor.OrganizationID.String(),
		},
	}, AgentDryRunWorkflowName, payload)
}

// TurnStart is what a worker needs to answer one turn.
type TurnStart struct {
	Actor   permissionports.RequestActor
	Content string
	Request AssistantTurnRequest
}

// StartTurnWorkflow hands a recorded turn to a worker and returns the
// execution carrying it.
//
// A person's question, a quick question and the application's own follow-up
// to a decision all start here, so they cannot drift apart in how a turn is
// handed over.
func StartTurnWorkflow(
	ctx context.Context,
	workflows kernelports.WorkflowStarter,
	turn *conversation.AssistantTurn,
	start TurnStart,
) (client.WorkflowRun, error) {
	origin := aitrace.Traceparent(ctx)
	anchored := aitrace.ContextWithAnchor(
		ctx,
		aitrace.AnchorFor(aitrace.AnchorAssistantTurn, turn.ID.String()),
	)

	return workflows.StartWorkflow(anchored, client.StartWorkflowOptions{
		ID:        WorkflowIDFor(turn.ID),
		TaskQueue: temporaltype.TaskQueueAgentChat.String(),
		// One turn, one execution. A duplicate start is a bug rather than a
		// second question, and rejecting it is how it stays visible.
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary:                            "Assistant turn",
		// Somebody is waiting on this one. Fairness by organization keeps one
		// busy Tenant from queueing everybody else's replies behind its own.
		Priority: temporal.Priority{
			PriorityKey: agentflowcontract.PriorityInteractive,
			FairnessKey: start.Actor.OrganizationID.String(),
		},
	}, AssistantTurnWorkflowName, &AssistantTurnPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: start.Actor.OrganizationID,
			BusinessUnitID: start.Actor.BusinessUnitID,
			UserID:         start.Actor.UserID,
			Timestamp:      timeutils.NowUnix(),
		},
		TurnID:   turn.ID,
		ThreadID: turn.ThreadID,
		Actor:    start.Actor,
		Content:  start.Content,
		Request:  start.Request,
		Origin:   origin,
	})
}

const AssistantTurnWorkflowName = "AssistantTurnWorkflow"

// WorkflowIDFor names the execution carrying a turn.
func WorkflowIDFor(turnID pulid.ID) string {
	return conversation.AssistantTurnWorkflowID(turnID)
}

// AssistantTurnPayload is one question and everything needed to answer it.
//
// The actor travels with the turn rather than being rebuilt on the worker.
// A turn runs as the person who asked and as nobody else, and a worker that
// reconstructed an actor from a Tenant would be inventing an authority that
// nobody granted.
type AssistantTurnPayload struct {
	temporaltype.BasePayload

	TurnID   pulid.ID                     `json:"turnId"`
	ThreadID pulid.ID                     `json:"threadId"`
	Actor    permissionports.RequestActor `json:"actor"`
	Content  string                       `json:"content"`
	Request  AssistantTurnRequest         `json:"request"`
	// Origin is the W3C traceparent of the request that started the turn,
	// so the turn's trace links back to it. Empty on a turn started before
	// it was kept, or from outside any trace.
	Origin string `json:"traceOrigin,omitempty"`
}

func (p *AssistantTurnPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.Actor.UserID,
	}
}

func (p *AssistantTurnPayload) SendRequest() *assistantports.SendMessageRequest {
	return &assistantports.SendMessageRequest{
		ThreadID:              p.ThreadID,
		Content:               p.Content,
		Page:                  p.Request.Page,
		Surface:               p.Request.Surface,
		TenantInfo:            p.TenantInfo(),
		PreferredProviderID:   p.Request.PreferredProviderID,
		ProviderChosen:        p.Request.ProviderChosen,
		AttachmentDocumentIDs: p.Request.AttachmentDocumentIDs,
		Mentions:              p.Request.Mentions,
		FollowUpProposalID:    p.Request.FollowUpProposalID,
		FollowUpPlanID:        p.Request.FollowUpPlanID,
		ResumeWaitID:          p.Request.ResumeWaitID,
		DirectedAgentID:       p.Request.DirectedAgentID,
		CodeMode:              p.Request.CodeMode,
	}
}

// AssistantTurnRequest is what the person handed over with the message.
type AssistantTurnRequest struct {
	Page                  *agent.PageContext `json:"page,omitempty"`
	Surface               agent.Surface      `json:"surface,omitempty"`
	Mentions              []agent.EntityRef  `json:"mentions,omitempty"`
	AttachmentDocumentIDs []pulid.ID         `json:"attachmentDocumentIds,omitempty"`
	PreferredProviderID   pulid.ID           `json:"preferredProviderId,omitempty"`
	ProviderChosen        bool               `json:"providerChosen"`
	FollowUpProposalID    pulid.ID           `json:"followUpProposalId,omitempty"`
	FollowUpPlanID        pulid.ID           `json:"followUpPlanId,omitempty"`
	// ResumeWaitID asks for the turn that picks up work the agent parked on
	// a wait, in place of content.
	ResumeWaitID pulid.ID `json:"resumeWaitId,omitempty"`
	// DirectedAgentID hands the message to another agent the person chose,
	// run from this conversation in place of its own agent's reply.
	DirectedAgentID pulid.ID `json:"directedAgentId,omitempty"`
	// Awaited says the request that asked is waiting for the turn's result
	// rather than reading its stream. That caller has the reply as soon as
	// the turn ends, so nobody is told later that it is ready.
	Awaited bool `json:"awaited,omitempty"`
	// CodeMode forces the code-mode gate for the turn; the desk bench sets it,
	// and only a development or test deployment honours "on".
	CodeMode codemode.Override `json:"codeMode,omitempty"`
}

// ReportsDecision says the turn was started to report a decision rather than
// to answer something the person asked.
func (r *AssistantTurnRequest) ReportsDecision() bool {
	return r.FollowUpProposalID.IsNotNil() || r.FollowUpPlanID.IsNotNil()
}

// ResumesWait says the turn was started to pick up work the agent parked on
// a wait rather than to answer something the person asked.
func (r *AssistantTurnRequest) ResumesWait() bool {
	return r.ResumeWaitID.IsNotNil()
}

// AssistantTurnResult is what the turn came to.
type AssistantTurnResult struct {
	Status  string `json:"status"`
	Refused bool   `json:"refused"`
	// Result is the saved turn, when it finished. A turn that failed or was
	// stopped has Message instead, written for the person who asked.
	Result  *assistantports.SendMessageResult `json:"result,omitempty"`
	Message string                            `json:"message,omitempty"`
}
