package paperworkjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/paperworkjobs/paperworkcontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func Workflows() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{{
		Name:        paperworkcontract.ReadWorkflowName,
		Fn:          PaperworkReadWorkflow,
		Description: "Nova reads a shipment document the moment it is uploaded",
	}}
}

// PaperworkReadWorkflow reads one document. A failed read retries a few times
// and then gives up; the document is still converted, and a person can ask
// Nova to read it again from the document review.
func PaperworkReadWorkflow(ctx workflow.Context, input *paperworkcontract.ReadInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2,
			MaximumAttempts:    3,
		},
	})

	var activities *Activities
	return workflow.ExecuteActivity(ctx, activities.ReadPaperworkActivity, input).Get(ctx, nil)
}
