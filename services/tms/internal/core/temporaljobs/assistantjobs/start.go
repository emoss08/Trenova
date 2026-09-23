package assistantjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// TurnStart is what a worker needs to answer one turn.
type TurnStart struct {
	Actor   serviceports.RequestActor
	Content string
	Request AssistantTurnRequest
}

// StartTurnWorkflow hands a recorded turn to a worker and returns the id of
// the execution carrying it.
//
// Both a person's question and the application's own follow-up to a decision
// start here, so the two cannot drift apart in how a turn is handed over.
func StartTurnWorkflow(
	ctx context.Context,
	workflows serviceports.WorkflowStarter,
	turn *conversation.AssistantTurn,
	start TurnStart,
) (string, error) {
	workflowID := WorkflowIDFor(turn.ID)

	_, err := workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: temporaltype.TaskQueueAgentChat.String(),
		// One turn, one execution. A duplicate start is a bug rather than
		// a second question, and rejecting it is how it stays visible.
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
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
	if err != nil {
		return "", err
	}

	return workflowID, nil
}
