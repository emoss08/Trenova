package triagewakejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var sweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        triagewakecontract.WakeDueWorkflowName,
			Fn:          WakeDueTriageWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Bring snoozed and waiting inbound messages back to the queue when their time comes",
		},
		{
			Name:        triagewakecontract.RecomputeDueWorkflowName,
			Fn:          RecomputeTriageDueWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Recompute due times after an organization changes its response targets",
		},
	}
}

// WakeDueTriageWorkflow is the timer behind every snoozed and waiting inbound
// message, run each minute: a parked message holds no workflow of its own.
func WakeDueTriageWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	var woken int
	err := workflow.ExecuteActivity(ctx, a.WakeDueTriageActivity).Get(ctx, &woken)

	return woken, err
}

func RecomputeTriageDueWorkflow(ctx workflow.Context, payload *RecomputeDuePayload) (int64, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	var changed int64
	err := workflow.ExecuteActivity(ctx, a.RecomputeTriageDueActivity, payload).Get(ctx, &changed)

	return changed, err
}
