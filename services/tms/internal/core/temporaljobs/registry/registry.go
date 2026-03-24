package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

type WorkerRegistry interface {
	GetName() string
	GetTaskQueue() string
	RegisterActivities(w worker.Worker) error
	RegisterWorkflows(w worker.Worker) error
	GetWorkerOptions() worker.Options
}

type WorkerConfig struct {
	MaxConcurrentActivityExecutionSize     int
	MaxConcurrentWorkflowTaskExecutionSize int
	MaxConcurrentWorkflowTaskPollers       int
	MaxConcurrentActivityTaskPollers       int
	EnableSessionWorker                    bool
	WorkerStopTimeout                      time.Duration
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
		MaxConcurrentWorkflowTaskPollers:       2,
		MaxConcurrentActivityTaskPollers:       2,
		EnableSessionWorker:                    true,
		WorkerStopTimeout:                      30 * time.Second,
	}
}

func (c WorkerConfig) ToWorkerOptions() worker.Options {
	return worker.Options{
		MaxConcurrentActivityExecutionSize:     c.MaxConcurrentActivityExecutionSize,
		MaxConcurrentWorkflowTaskExecutionSize: c.MaxConcurrentWorkflowTaskExecutionSize,
		MaxConcurrentWorkflowTaskPollers:       c.MaxConcurrentWorkflowTaskPollers,
		MaxConcurrentActivityTaskPollers:       c.MaxConcurrentActivityTaskPollers,
		EnableSessionWorker:                    c.EnableSessionWorker,
		WorkerStopTimeout:                      c.WorkerStopTimeout,
	}
}

type WorkerManager struct {
	client     client.Client
	workers    map[string]worker.Worker
	registries []WorkerRegistry
	logger     *zap.Logger
	mu         sync.RWMutex
}

func NewWorkerManager(c client.Client, logger *zap.Logger) *WorkerManager {
	return &WorkerManager{
		client:     c,
		workers:    make(map[string]worker.Worker),
		registries: make([]WorkerRegistry, 0),
		logger:     logger.Named("worker-manager"),
	}
}

func (m *WorkerManager) Register(registry WorkerRegistry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := registry.GetName()
	if _, exists := m.workers[name]; exists {
		return fmt.Errorf("worker %s already registered", name)
	}

	taskQueue := registry.GetTaskQueue()
	if m.client == nil {
		m.logger.Warn("cannot register worker: temporal client is not configured",
			zap.String("name", name),
			zap.String("taskQueue", taskQueue),
		)
		return fmt.Errorf("%w: worker=%s", ErrTemporalClientNotConfigured, name)
	}

	m.logger.Info("registering worker",
		zap.String("name", name),
		zap.String("taskQueue", taskQueue),
	)

	w := worker.New(m.client, taskQueue, registry.GetWorkerOptions())

	if err := registry.RegisterActivities(w); err != nil {
		return fmt.Errorf("failed to register activities for %s: %w", name, err)
	}

	if err := registry.RegisterWorkflows(w); err != nil {
		return fmt.Errorf("failed to register workflows for %s: %w", name, err)
	}

	m.workers[name] = w
	m.registries = append(m.registries, registry)

	m.logger.Info("worker registered successfully",
		zap.String("name", name),
		zap.String("taskQueue", taskQueue),
	)

	return nil
}

func (m *WorkerManager) StartAll(_ context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.workers) == 0 {
		return ErrNoWorkersRegistered
	}

	m.logger.Info("starting all workers", zap.Int("count", len(m.workers)))

	var startErrors []error
	for name, w := range m.workers {
		m.logger.Info("starting worker", zap.String("name", name))

		if err := w.Start(); err != nil {
			m.logger.Error("failed to start worker",
				zap.String("name", name),
				zap.Error(err),
			)
			startErrors = append(startErrors, fmt.Errorf("start worker %s: %w", name, err))
		} else {
			m.logger.Info("worker started successfully", zap.String("name", name))
		}
	}

	if len(startErrors) > 0 {
		return fmt.Errorf("failed to start %d worker(s): %v", len(startErrors), startErrors)
	}

	m.logger.Info("all workers started successfully")
	return nil
}

func (m *WorkerManager) StopAll(_ context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.logger.Info("stopping all workers", zap.Int("count", len(m.workers)))

	for name, w := range m.workers {
		m.logger.Info("stopping worker", zap.String("name", name))
		w.Stop()
	}

	m.logger.Info("all workers stopped")
	return nil
}

func (m *WorkerManager) GetWorker(name string) (worker.Worker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w, exists := m.workers[name]
	return w, exists
}

type DomainConfig struct {
	Name         string
	TaskQueue    string
	WorkerConfig WorkerConfig
}

type WorkflowDefinition struct {
	Name        string
	Fn          any
	Description string
}

type DomainRegistry struct {
	config     *DomainConfig
	activities any
	workflows  []WorkflowDefinition
	logger     *zap.Logger
}

func NewDomainRegistry(
	config *DomainConfig,
	activities any,
	workflows []WorkflowDefinition,
	logger *zap.Logger,
) *DomainRegistry {
	return &DomainRegistry{
		config:     config,
		activities: activities,
		workflows:  workflows,
		logger:     logger.Named(config.Name + "-registry"),
	}
}

func (r *DomainRegistry) GetName() string {
	return r.config.Name
}

func (r *DomainRegistry) GetTaskQueue() string {
	return r.config.TaskQueue
}

func (r *DomainRegistry) RegisterActivities(w worker.Worker) error {
	w.RegisterActivity(r.activities)
	r.logger.Info("registered activities struct")
	return nil
}

func (r *DomainRegistry) RegisterWorkflows(w worker.Worker) error {
	for _, wf := range r.workflows {
		w.RegisterWorkflow(wf.Fn)
		r.logger.Debug("registered workflow",
			zap.String("name", wf.Name),
			zap.String("description", wf.Description),
		)
	}
	r.logger.Info("registered workflows", zap.Int("count", len(r.workflows)))
	return nil
}

func (r *DomainRegistry) GetWorkerOptions() worker.Options {
	return r.config.WorkerConfig.ToWorkerOptions()
}
