package desktranscriptjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	CleanupScheduleID = "desk-transcript-export-cleanup"
	cleanupEvery      = time.Hour
)

var DomainConfig = registry.DomainConfig{
	Name:         "desk-transcript-export-worker",
	TaskQueue:    temporaltype.ReportTaskQueue,
	WorkerConfig: reportjobs.DomainConfig.WorkerConfig,
}

func convertWorkflows(wfs []temporaltype.WorkflowDefinition) []registry.WorkflowDefinition {
	result := make([]registry.WorkflowDefinition, len(wfs))
	for i, wf := range wfs {
		result[i] = registry.WorkflowDefinition{
			Name:        wf.Name,
			Fn:          wf.Fn,
			Description: wf.Description,
		}
	}

	return result
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
		Workflows:  convertWorkflows(RegisterWorkflows()),
		Logger:     p.Logger,
	})
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            CleanupScheduleID,
			Feature:       platformcatalog.FeatureAIDesk,
			Description:   "Delete Desk conversation exports past their download window",
			Spec:          schedule.Every(cleanupEvery),
			Workflow:      CleanupDeskTranscriptExportsWorkflow,
			TaskQueue:     temporaltype.ReportTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "desk-transcript-export-cleanup",
				"target":  "desk_transcript_exports",
			},
		},
	}
}

var Module = fx.Module("desk-transcript-jobs",
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
