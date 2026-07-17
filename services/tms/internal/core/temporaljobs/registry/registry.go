package registry

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
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

type QueueFilter struct {
	Queues []string
}

func (f *QueueFilter) Allows(taskQueue string) bool {
	if f == nil || len(f.Queues) == 0 {
		return true
	}
	for _, q := range f.Queues {
		if q == taskQueue {
			return true
		}
	}
	return false
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
		EnableSessionWorker:                    false,
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
	client       client.Client
	workers      map[string]worker.Worker
	queueWorkers map[string]worker.Worker
	queueOptions map[string]worker.Options
	registries   []WorkerRegistry
	interceptors []interceptor.WorkerInterceptor
	logger       *zap.Logger
	mu           sync.RWMutex
}

func (m *WorkerManager) SetInterceptors(interceptors []interceptor.WorkerInterceptor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interceptors = interceptors
}

func NewWorkerManager(c client.Client, logger *zap.Logger) *WorkerManager {
	return &WorkerManager{
		client:       c,
		workers:      make(map[string]worker.Worker),
		queueWorkers: make(map[string]worker.Worker),
		queueOptions: make(map[string]worker.Options),
		registries:   make([]WorkerRegistry, 0),
		logger:       logger.Named("worker-manager"),
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

	opts := registry.GetWorkerOptions()
	if len(m.interceptors) > 0 {
		opts.Interceptors = append(opts.Interceptors, m.interceptors...)
	}
	w, exists := m.queueWorkers[taskQueue]
	if exists && !workerOptionsEqual(m.queueOptions[taskQueue], opts) {
		return fmt.Errorf(
			"worker options for task queue %q are incompatible with registry %q",
			taskQueue,
			name,
		)
	}

	m.logger.Info("registering worker",
		zap.String("name", name),
		zap.String("taskQueue", taskQueue),
		zap.Bool("sharedTaskQueue", exists),
	)

	createdQueueWorker := false
	if !exists {
		w = worker.New(m.client, taskQueue, opts)
		createdQueueWorker = true
	}

	if err := registry.RegisterActivities(w); err != nil {
		return fmt.Errorf("failed to register activities for %s: %w", name, err)
	}

	if err := registry.RegisterWorkflows(w); err != nil {
		return fmt.Errorf("failed to register workflows for %s: %w", name, err)
	}

	if createdQueueWorker {
		m.queueWorkers[taskQueue] = w
		m.queueOptions[taskQueue] = opts
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

	if len(m.queueWorkers) == 0 {
		return ErrNoWorkersRegistered
	}

	m.logger.Info("starting all workers", zap.Int("count", len(m.queueWorkers)))

	var startErrors []error
	for taskQueue, w := range m.queueWorkers {
		m.logger.Info("starting worker", zap.String("taskQueue", taskQueue))

		if err := w.Start(); err != nil {
			m.logger.Error("failed to start worker",
				zap.String("taskQueue", taskQueue),
				zap.Error(err),
			)
			startErrors = append(startErrors, fmt.Errorf("start worker for task queue %s: %w", taskQueue, err))
		} else {
			m.logger.Info("worker started successfully", zap.String("taskQueue", taskQueue))
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

	m.logger.Info("stopping all workers", zap.Int("count", len(m.queueWorkers)))

	for taskQueue, w := range m.queueWorkers {
		m.logger.Info("stopping worker", zap.String("taskQueue", taskQueue))
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

func workerOptionsEqual(left, right worker.Options) bool {
	return reflect.DeepEqual(left, right)
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
