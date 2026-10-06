package conversationschedulejobs

import (
	"context"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// reconcileEvery is how often every conversation schedule is checked against
// its row. A save syncs its own schedule at once and a run whose schedule is
// gone removes it; this repairs what a failed call left behind.
const reconcileEvery = time.Hour

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            ReconcileScheduleID,
			Description:   "Keep one schedule behind every request scheduled in a conversation",
			Spec:          schedule.Every(reconcileEvery),
			Workflow:      ReconcileConversationSchedulesWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ReconcileScheduleID,
			},
		},
	}
}

var Module = fx.Module("conversation-schedule-jobs",
	fx.Provide(NewActivities),
	fx.Provide(fx.Annotate(
		NewSchedules,
		fx.As(fx.Self()),
		fx.As(new(serviceports.ConversationScheduleSyncer)),
	)),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)

// WorkerModule reconciles every conversation schedule as a worker starts, so
// one saved while no worker ran fires on time rather than an hour later.
var WorkerModule = fx.Module("conversation-schedule-jobs-worker",
	fx.Invoke(reconcileOnStart),
)

// reconcileOnStartID names the reconcile a starting worker asks for. Workers
// starting together ask for the same execution, so it runs once.
const reconcileOnStartID = "conversation-schedules-on-start"

func reconcileOnStart(lc fx.Lifecycle, c client.Client, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if c == nil {
				return nil
			}
			_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				ID:                       reconcileOnStartID,
				TaskQueue:                DomainConfig.TaskQueue,
				WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			}, ReconcileWorkflowName)
			if err != nil {
				// The hourly reconcile covers it; a worker that cannot ask is
				// not a worker that cannot start.
				logger.Warn("could not start reconciling conversation schedules", zap.Error(err))
			}

			return nil
		},
	})
}
