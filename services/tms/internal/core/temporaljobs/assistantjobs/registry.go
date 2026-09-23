package assistantjobs

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DomainConfig puts interactive turns on their own queue.
//
// It is the reason the queue exists: a person watching a reply arrive must not
// wait behind a scheduled run that takes ten minutes. The worker is shaped for
// that — many turns at once, each one long and mostly idle, waiting on a model
// rather than on a CPU.
var DomainConfig = registry.DomainConfig{
	Name:      "assistant-worker",
	TaskQueue: temporaltype.TaskQueueAgentChat.String(),
	WorkerConfig: registry.WorkerConfig{
		MaxConcurrentActivityExecutionSize:     100,
		MaxConcurrentWorkflowTaskExecutionSize: 50,
		MaxConcurrentWorkflowTaskPollers:       4,
		MaxConcurrentActivityTaskPollers:       8,
		WorkerStopTimeout:                      registry.DefaultWorkerConfig().WorkerStopTimeout,
	},
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Flow       *agentflow.Activities
	Workflows  *Workflows
	Logger     *zap.Logger
}

// Registry puts a turn's workflow and every activity it runs on the chat
// queue: its own first and last steps, the model call, the tool search, and
// every tool as the worker's dynamic activity.
type Registry struct {
	activities *Activities
	flow       *agentflow.Activities
	workflows  *Workflows
	logger     *zap.Logger
}

var (
	_ registry.WorkerRegistry = (*Registry)(nil)
	_ registry.ActivityNamer  = (*Registry)(nil)
)

func NewRegistry(p RegistryParams) *Registry {
	return &Registry{
		activities: p.Activities,
		flow:       p.Flow,
		workflows:  p.Workflows,
		logger:     p.Logger.Named(DomainConfig.Name + "-registry"),
	}
}

func (r *Registry) GetName() string { return DomainConfig.Name }

func (r *Registry) GetTaskQueue() string { return DomainConfig.TaskQueue }

func (r *Registry) GetWorkerOptions() worker.Options {
	return DomainConfig.WorkerConfig.ToWorkerOptions()
}

func (r *Registry) RegisterActivities(w worker.Worker) error {
	w.RegisterActivity(r.activities)
	agentflow.RegisterActivities(w, r.flow)
	r.logger.Info("registered assistant turn activities")

	return nil
}

func (r *Registry) RegisterWorkflows(w worker.Worker) error {
	w.RegisterWorkflowWithOptions(r.workflows.AssistantTurnWorkflow, workflow.RegisterOptions{
		Name: AssistantTurnWorkflowName,
	})

	return nil
}

func (r *Registry) ActivityNames() []string {
	names := append(
		registry.ActivityNamesOf(r.activities),
		registry.ActivityNamesOf(r.flow)...,
	)
	slices.Sort(names)

	return names
}
