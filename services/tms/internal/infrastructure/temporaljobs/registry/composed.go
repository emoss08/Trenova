package registry

import (
	"slices"

	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

// ComposedParams describe a queue whose worker registers more than one
// activities struct, or registers something a struct cannot hold, such as a
// dynamic activity.
type ComposedParams struct {
	Config *DomainConfig
	// Activities are structs whose activity methods are registered by name.
	Activities []any
	// Register registers whatever else the queue's activities need.
	Register func(w worker.ActivityRegistry)
	// Workflows are registered under their Name, so a method value or a
	// closure is registered under the name the starter uses rather than
	// whatever the SDK would derive from the function.
	Workflows []WorkflowDefinition
	Logger    *zap.Logger
}

// ComposedRegistry registers a queue's work from several sources.
type ComposedRegistry struct {
	p      ComposedParams
	logger *zap.Logger
}

var (
	_ WorkerRegistry = (*ComposedRegistry)(nil)
	_ ActivityNamer  = (*ComposedRegistry)(nil)
)

func NewComposedRegistry(p ComposedParams) *ComposedRegistry {
	return &ComposedRegistry{p: p, logger: p.Logger.Named(p.Config.Name + "-registry")}
}

func (r *ComposedRegistry) GetName() string { return r.p.Config.Name }

func (r *ComposedRegistry) GetTaskQueue() string { return r.p.Config.TaskQueue }

func (r *ComposedRegistry) GetWorkerOptions() worker.Options {
	return r.p.Config.WorkerConfig.ToWorkerOptions()
}

func (r *ComposedRegistry) RegisterActivities(w worker.Worker) error {
	for _, activities := range r.p.Activities {
		w.RegisterActivity(activities)
	}
	if r.p.Register != nil {
		r.p.Register(w)
	}
	r.logger.Info("registered activities", zap.Int("structs", len(r.p.Activities)))

	return nil
}

func (r *ComposedRegistry) RegisterWorkflows(w worker.Worker) error {
	for _, wf := range r.p.Workflows {
		w.RegisterWorkflowWithOptions(wf.Fn, workflow.RegisterOptions{Name: wf.Name})
		r.logger.Debug("registered workflow",
			zap.String("name", wf.Name),
			zap.String("description", wf.Description),
		)
	}
	r.logger.Info("registered workflows", zap.Int("count", len(r.p.Workflows)))

	return nil
}

func (r *ComposedRegistry) ActivityNames() []string {
	names := make([]string, 0, len(r.p.Activities)*8)
	for _, activities := range r.p.Activities {
		names = append(names, ActivityNamesOf(activities)...)
	}
	slices.Sort(names)

	return names
}
