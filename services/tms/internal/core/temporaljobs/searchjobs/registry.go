package searchjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
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
		logger:     p.Logger.Named("search-worker-registry"),
		config:     registry.DefaultWorkerConfig(),
	}
}

func (r *Registry) GetName() string {
	return "search-worker"
}

func (r *Registry) GetTaskQueue() string {
	return SearchTaskQueue
}

func (r *Registry) RegisterActivities(w worker.Worker) error {
	w.RegisterActivity(r.activities.IndexEntityActivity)
	w.RegisterActivity(r.activities.BulkIndexEntityActivity)

	r.logger.Info("registered search activities",
		zap.Int("count", 1),
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

	r.logger.Info("registered search workflows",
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
