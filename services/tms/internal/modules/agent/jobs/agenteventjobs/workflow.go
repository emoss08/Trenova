package agenteventjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agenteventjobs/agenteventcontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// drainDebounce is also how often a drain waiting for room looks again.
	drainDebounce         = 15 * time.Second
	drainPagesPerRound    = 5
	drainIterationsPerRun = 50
)

var startOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:        agenteventcontract.DrainWorkflowName,
		Fn:          AgentEventDrainWorkflow,
		TaskQueue:   temporaltype.TaskQueueAgentBackground.String(),
		Description: "Start an agent's queued event runs, as many as it may run at once",
	}}
}

// AgentEventDrainWorkflow starts one agent's queued event runs in a tenant,
// never more working at once than the agent allows. While the agent is full
// it looks again each debounce, and ends once its queue is empty.
func AgentEventDrainWorkflow(
	ctx workflow.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainTotals, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, startOptions)
	totals, again, err := temporaljobs.RunTenantDrain(ctx, temporaljobs.TenantDrainOptions{
		SignalName:       agenteventcontract.DrainSignalName,
		Debounce:         drainDebounce,
		PagesPerRound:    drainPagesPerRound,
		IterationsPerRun: drainIterationsPerRun,
		Iterations:       payload.Iterations,
		Page: func(ctx workflow.Context) (*temporaljobs.DrainPage, error) {
			var page temporaljobs.DrainPage
			err := workflow.ExecuteActivity(actx, a.StartEventRunsActivity, payload).
				Get(actx, &page)

			return &page, err
		},
	})
	if err != nil || !again {
		return totals, err
	}

	return totals, workflow.NewContinueAsNewError(ctx, AgentEventDrainWorkflow,
		&temporaljobs.DrainPayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			Kind:           payload.Kind,
		})
}
