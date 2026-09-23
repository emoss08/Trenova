package agentjobs

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// asRegistry wires one of the agent's registries into the worker group. Which
// of them a given process actually runs is decided by --queues at startup, not
// here: every worker binary knows about all three and polls only the queues it
// was told to.
func asRegistry(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(registry.WorkerRegistry)),
		fx.ResultTags(`group:"worker_registries"`),
	)
}

var Module = fx.Module("agent-jobs",
	fx.Provide(NewActivities, NewWorkflows),
	fx.Provide(fx.Annotate(
		NewDefinitionSchedules,
		fx.As(fx.Self()),
		fx.As(new(serviceports.AgentDefinitionScheduler)),
	)),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		asRegistry(NewBackgroundRegistry),
		asRegistry(NewHeavyRegistry),
		asRegistry(NewDrainRegistry),
	),
)

// WorkerModule reconciles every agent's schedule as a worker starts, so an
// agent saved before its schedule existed, or while no worker ran, runs on
// time rather than a quarter hour later.
var WorkerModule = fx.Module("agent-jobs-worker",
	fx.Invoke(reconcileOnStart),
)

// reconcileOnStartID names the reconcile a starting worker asks for. Workers
// starting together ask for the same execution, so it runs once.
const reconcileOnStartID = "agent-definition-schedules-on-start"

func reconcileOnStart(lc fx.Lifecycle, c client.Client, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				ID:                       reconcileOnStartID,
				TaskQueue:                BackgroundDomainConfig.TaskQueue,
				WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			}, ReconcileSchedulesWorkflowName)
			if err != nil {
				// The quarter-hourly reconcile covers it; a worker that cannot
				// ask is not a worker that cannot start.
				logger.Warn("could not start reconciling agent schedules", zap.Error(err))
			}

			return nil
		},
	})
}
