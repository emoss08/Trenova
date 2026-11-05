package scimjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SweepScheduleID   = "scim-pending-grants-sweep"
	SweepWorkflowName = "SweepPendingSCIMGrantsWorkflow"
	sweepEvery        = 15 * time.Minute
)

var DomainConfig = registry.DomainConfig{
	Name:         "identity-scim-worker",
	TaskQueue:    temporaltype.TaskQueueSystem.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Module = fx.Module("identity-scim-jobs",
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

func workflows() []registry.WorkflowDefinition {
	defs := RegisterWorkflows()
	out := make([]registry.WorkflowDefinition, len(defs))
	for i, wf := range defs {
		out[i] = registry.WorkflowDefinition{Name: wf.Name, Fn: wf.Fn, Description: wf.Description}
	}

	return out
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, workflows(), p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID: SweepScheduleID,
		Description: "Start a reconcile for every SCIM directory whose group roles still wait " +
			"to be brought up to date",
		Spec:          schedule.Every(sweepEvery),
		Workflow:      SweepPendingSCIMGrantsWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": SweepScheduleID},
	}}
}
