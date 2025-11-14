package reportjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

type Registry struct {
	activities *Activities
	logger     *zap.Logger
	config     registry.WorkerConfig
}

func NewRegistry(p RegistryParams) *Registry {
	return &Registry{
		activities: p.Activities,
		logger:     p.Logger.Named("report-worker-registry"),
		config:     registry.DefaultWorkerConfig(),
	}
}

func (r *Registry) GetName() string {
	return "report-worker"
}

func (r *Registry) GetTaskQueue() string {
	return temporaltype.ReportTaskQueue
}

func (r *Registry) RegisterActivities(w worker.Worker) error {
	w.RegisterActivity(r.activities.UpdateReportStatusActivity)
	w.RegisterActivity(r.activities.ExecuteQueryActivity)
	w.RegisterActivity(r.activities.GenerateFileActivity)
	w.RegisterActivity(r.activities.UploadToStorageActivity)
	w.RegisterActivity(r.activities.UpdateReportCompletedActivity)
	w.RegisterActivity(r.activities.MarkReportFailedActivity)
	w.RegisterActivity(r.activities.SendReportEmailActivity)
	w.RegisterActivity(r.activities.SendReportReadyNotificationActivity)

	r.logger.Info("registered report activities",
		zap.Int("count", 8),
	)

	return nil
}

func (r *Registry) RegisterWorkflows(w worker.Worker) error {
	workflows := RegisterWorkflows()

	for _, wf := range workflows {
		w.RegisterWorkflow(wf.Fn)
		r.logger.Info("registered workflow",
			zap.String("name", wf.Name),
			zap.String("description", wf.Description),
		)
	}

	r.logger.Info("registered report workflows",
		zap.Int("count", len(workflows)),
	)

	return nil
}

func (r *Registry) GetWorkerOptions() worker.Options {
	return r.config.ToWorkerOptions()
}

func (r *Registry) WithConfig(config registry.WorkerConfig) *Registry {
	r.config = config
	return r
}
