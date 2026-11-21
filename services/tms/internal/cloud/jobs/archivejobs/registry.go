package archivejobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const NightlyScheduleID = "tenant-archive-nightly"

var DomainConfig = registry.DomainConfig{
	Name:         "cloud-tenant-archive-worker",
	TaskQueue:    temporaltype.TaskQueueSystem.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

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

type ScheduleProvider struct {
	managed *managedconfig.Config
}

func NewScheduleProvider(managed *managedconfig.Config) *ScheduleProvider {
	return &ScheduleProvider{managed: managed}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	if !p.managed.SharedTenancy() {
		return []*schedule.Schedule{}
	}
	archive := &p.managed.Archive
	return []*schedule.Schedule{
		{
			ID:          NightlyScheduleID,
			Feature:     platformcatalog.FeatureAdministration,
			Description: "Archive each active shared-server tenant once a night, off-peak, and prune its archives past retention",
			Spec:        schedule.Cron(archive.NightlyCron()),
			Workflow:    CloudTenantNightlyArchiveWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Args: []any{NightlyArgs{
				Window:      archive.Window(),
				Concurrency: archive.Workers(),
			}},
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
	}
}

var Module = fx.Module("cloud-tenant-archive-jobs",
	fx.Provide(NewActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
)
