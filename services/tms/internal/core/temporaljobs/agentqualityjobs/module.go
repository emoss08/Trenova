package agentqualityjobs

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var DomainConfig = registry.DomainConfig{
	Name:         "agent-quality-worker",
	TaskQueue:    temporaltype.TaskQueueAgentHeavy.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

func workflows() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{
		{
			Name:        AgentQualitySweepWorkflowName,
			Fn:          AgentQualitySweepWorkflow,
			Description: "Plan an organization's nightly agent quality sweep and run each changed agent's suite",
		},
		{
			Name:        AgentSuiteRunWorkflowName,
			Fn:          AgentSuiteRunWorkflow,
			Description: "Replay one agent's sampled evaluation cases within budget, score them and finalize the run",
		},
		{
			Name:        ReconcileQualitySchedulesWorkflowName,
			Fn:          ReconcileQualitySchedulesWorkflow,
			Description: reconcileDescription,
		},
	}
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     &DomainConfig,
		Activities: []any{p.Activities},
		Workflows:  workflows(),
		Logger:     p.Logger,
	})
}

var Module = fx.Module("agent-quality-jobs",
	fx.Provide(
		NewActivities,
		NewQualitySchedules,
		func(s *QualitySchedules) serviceports.AgentQualityScheduler { return s },
		func(s *QualitySchedules) serviceports.AgentSuiteStarter { return s },
	),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(fx.Annotate(
		NewRegistry,
		fx.As(new(registry.WorkerRegistry)),
		fx.ResultTags(`group:"worker_registries"`),
	)),
)

var WorkerModule = fx.Module("agent-quality-jobs-worker",
	fx.Invoke(reconcileOnStart),
)

func reconcileOnStart(lc fx.Lifecycle, c client.Client, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				ID:                       reconcileQualitySchedulesOnStartWorkID,
				TaskQueue:                DomainConfig.TaskQueue,
				WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			}, ReconcileQualitySchedulesWorkflowName)
			if err != nil {
				logger.Warn("could not start reconciling quality sweep schedules", zap.Error(err))
			}

			return nil
		},
	})
}
