package conversationschedulejobs

import (
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// fireOptions bound starting one scheduled turn. The retries are mostly for a
// conversation busy with another reply when the slot came: they wait it out
// for several minutes rather than drop the run, and give up before the next
// slot could be due.
var fireOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    30 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    2 * time.Minute,
		MaximumAttempts:    6,
	},
}

// reconcileOptions bound one pass over every conversation schedule.
var reconcileOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        RunWorkflowName,
			Fn:          ConversationScheduleRunWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Ask a scheduled request in the conversation it was made in",
		},
		{
			Name:        ReconcileWorkflowName,
			Fn:          ReconcileConversationSchedulesWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Keep one Temporal schedule behind every conversation schedule",
		},
	}
}

// ConversationScheduleRunWorkflow is one slot of a conversation schedule. It
// hands the slot to the service, which starts the turn; the turn is its own
// workflow, so this one ends as soon as the turn has begun.
func ConversationScheduleRunWorkflow(
	ctx workflow.Context,
	payload *RunPayload,
) (*serviceports.FireConversationScheduleResult, error) {
	var a *Activities

	fireCtx := workflow.WithActivityOptions(ctx, fireOptions)
	var result serviceports.FireConversationScheduleResult
	err := workflow.ExecuteActivity(fireCtx, a.FireConversationScheduleActivity, &FireInput{
		Payload: payload,
		FiredAt: workflow.GetInfo(ctx).WorkflowStartTime.Unix(),
	}).Get(fireCtx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ReconcileConversationSchedulesWorkflow makes every conversation schedule's
// Temporal schedule match its row. It runs when a worker starts and hourly.
func ReconcileConversationSchedulesWorkflow(ctx workflow.Context) (*ReconcileResult, error) {
	var a *Activities

	reconcileCtx := workflow.WithActivityOptions(ctx, reconcileOptions)
	var result ReconcileResult
	if err := workflow.ExecuteActivity(reconcileCtx, a.ReconcileConversationSchedulesActivity).
		Get(reconcileCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
