package podchasejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/podchasejobs/podchasecontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	drainDebounce         = 15 * time.Second
	drainPagesPerRound    = 10
	drainIterationsPerRun = 25
)

var chaseOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        podchasecontract.DrainWorkflowName,
			Fn:          PODChaseDrainWorkflow,
			TaskQueue:   temporaltype.TaskQueueAgentBackground.String(),
			Description: "Ask for the proofs of delivery a tenant's delivered shipments are missing, a page at a time",
		},
	}
}

// PODChaseDrainWorkflow asks for a tenant's queued proofs of delivery shortly
// after the shipments are found missing them: one workflow per tenant, and
// no model call for a shipment someone can be asked about.
func PODChaseDrainWorkflow(
	ctx workflow.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainTotals, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, chaseOptions)
	totals, again, err := temporaljobs.RunTenantDrain(ctx, temporaljobs.TenantDrainOptions{
		SignalName:       podchasecontract.DrainSignalName,
		Debounce:         drainDebounce,
		PagesPerRound:    drainPagesPerRound,
		IterationsPerRun: drainIterationsPerRun,
		Iterations:       payload.Iterations,
		Page: func(ctx workflow.Context) (*temporaljobs.DrainPage, error) {
			var page temporaljobs.DrainPage
			err := workflow.ExecuteActivity(actx, a.ChasePODsActivity, payload).Get(actx, &page)

			return &page, err
		},
	})
	if err != nil || !again {
		return totals, err
	}

	return totals, workflow.NewContinueAsNewError(ctx, PODChaseDrainWorkflow,
		&temporaljobs.DrainPayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			Kind:           payload.Kind,
		})
}
