package dispatchlearningjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const learnDescription = "Learn each organization's dispatch style from the assignments its dispatchers made"

var DomainConfig = registry.DomainConfig{
	Name:         "dispatch-learning-worker",
	TaskQueue:    temporaltype.TaskQueueDispatch.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        LearnDispatchStyleWorkflowName,
		Fn:          LearnDispatchStyleWorkflow,
		Description: learnDescription,
	},
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, Workflows, p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            "dispatch-style-learning",
			Description:   learnDescription,
			Spec:          schedule.Every(15 * time.Minute),
			Workflow:      LearnDispatchStyleWorkflow,
			Args:          []any{&FanOutInput{}},
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "dispatch-style-learning",
			},
		},
	}
}

var Module = fx.Module("dispatch-learning-jobs",
	fx.Provide(NewActivities),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
