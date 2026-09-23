package assistantjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// TurnStart is what a worker needs to answer one turn.
type TurnStart struct {
	Actor   serviceports.RequestActor
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
	workflows serviceports.WorkflowStarter,
	turn *conversation.AssistantTurn,
	start TurnStart,
) (client.WorkflowRun, error) {
	return workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:        WorkflowIDFor(turn.ID),
		TaskQueue: temporaltype.TaskQueueAgentChat.String(),
		// One turn, one execution. A duplicate start is a bug rather than a
		// second question, and rejecting it is how it stays visible.
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary:                            "Assistant turn",
		// Somebody is waiting on this one. Fairness by organization keeps one
		// busy tenant from queueing everybody else's replies behind its own.
		Priority: temporal.Priority{
			PriorityKey: agentflow.PriorityInteractive,
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
	})
}
