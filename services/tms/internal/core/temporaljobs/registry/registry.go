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

	for name, w := range m.workers {
		workerInstance := w
		workerName := name

		go func() {
			m.logger.Info("starting worker", zap.String("name", workerName))

			if err := workerInstance.Run(worker.InterruptCh()); err != nil {
				m.logger.Error("worker stopped with error",
					zap.String("name", workerName),
					zap.Error(err),
				)
			} else {
				m.logger.Info("worker stopped gracefully",
					zap.String("name", workerName),
				)
			}
		}()
	}

	time.Sleep(1 * time.Second)
	m.logger.Info("all workers started")

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
