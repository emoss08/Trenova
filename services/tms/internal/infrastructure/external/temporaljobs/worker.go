/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
)

// WorkerParams defines dependencies for the worker
type WorkerParams struct {
	fx.In

	Logger *logger.Logger
	Client client.Client
}

// Worker manages Temporal workers for different task queues
type Worker struct {
	client     client.Client
	logger     *zerolog.Logger
	workers    map[string]worker.Worker
	mu         sync.RWMutex
	isRunning  atomic.Bool
	startTime  time.Time
	panicCount atomic.Int32
	lastPanic  atomic.Pointer[time.Time]
	
	// Activity providers to register after workers are created
	activityProviders []any
}

// NewWorker creates a new Temporal worker manager
func NewWorker(p WorkerParams) *Worker {
	log := p.Logger.With().
		Str("component", "temporal-worker").
		Logger()

	return &Worker{
		client:  p.Client,
		logger:  &log,
		workers: make(map[string]worker.Worker),
	}
}

// RegisterWorkflow registers a workflow with all workers
func (w *Worker) RegisterWorkflow(workflowFunc any) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, wrk := range w.workers {
		wrk.RegisterWorkflow(workflowFunc)
	}
}

// RegisterWorkflowWithOptions registers a workflow with specific options
func (w *Worker) RegisterWorkflowWithOptions(workflowFunc any, options workflow.RegisterOptions) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, wrk := range w.workers {
		wrk.RegisterWorkflowWithOptions(workflowFunc, options)
	}
}

// RegisterActivity registers an activity with all workers
func (w *Worker) RegisterActivity(activityFunc any) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, wrk := range w.workers {
		wrk.RegisterActivity(activityFunc)
	}
}

// Start initializes and starts workers for all task queues
func (w *Worker) Start() error {
	w.logger.Info().Msg("starting temporal workers")

	taskQueues := map[string]int{
		TaskQueueCritical:   10, // 50% of workers
		TaskQueueEmail:      4,  // 20% of workers
		TaskQueueShipment:   2,  // 10% of workers
		TaskQueuePattern:    2,  // 10% of workers
		TaskQueueCompliance: 2,  // 10% of workers
		TaskQueueDefault:    2,  // 10% of workers
	}

	for taskQueue, maxConcurrency := range taskQueues {
		workflowConcurrency := max(maxConcurrency, 2)

		wrk := worker.New(w.client, taskQueue, worker.Options{
			MaxConcurrentActivityExecutionSize:      maxConcurrency,
			MaxConcurrentWorkflowTaskExecutionSize:  workflowConcurrency,
			MaxConcurrentLocalActivityExecutionSize: maxConcurrency,
			EnableLoggingInReplay:                   false,
		})

		w.workers[taskQueue] = wrk

		w.logger.Info().
			Str("task_queue", taskQueue).
			Int("max_concurrency", maxConcurrency).
			Msg("created worker for task queue")
	}

	w.logger.Info().Msg("registering workflows and activities")
	RegisterWorkflowsAndActivities(w)
	
	// Register activity providers on the appropriate workers
	for _, provider := range w.activityProviders {
		// Register on the shipment task queue worker specifically
		if shipmentWorker, ok := w.workers[TaskQueueShipment]; ok {
			shipmentWorker.RegisterActivity(provider)
			w.logger.Info().
				Str("task_queue", TaskQueueShipment).
				Str("provider", fmt.Sprintf("%T", provider)).
				Msg("registered activity provider on worker")
		}
	}

	w.isRunning.Store(true)
	w.startTime = time.Now()

	for taskQueue, wrk := range w.workers {
		go w.runWorkerWithRecovery(taskQueue, wrk)
	}

	w.logger.Info().
		Int("worker_count", len(w.workers)).
		Msg("temporal workers started successfully")

	return nil
}

// runWorkerWithRecovery runs a worker with panic recovery
func (w *Worker) runWorkerWithRecovery(taskQueue string, wrk worker.Worker) {
	defer func() {
		if r := recover(); r != nil {
			now := time.Now()
			w.panicCount.Add(1)
			w.lastPanic.Store(&now)

			w.logger.Error().
				Str("task_queue", taskQueue).
				Interface("panic", r).
				Int32("total_panics", w.panicCount.Load()).
				Msg("worker panicked - attempting restart")

			// Wait before restarting
			time.Sleep(5 * time.Second)

			// Only restart if we haven't had too many panics
			if w.panicCount.Load() < 10 {
				go w.runWorkerWithRecovery(taskQueue, wrk)
			} else {
				w.logger.Error().
					Str("task_queue", taskQueue).
					Msg("worker exceeded maximum panic count - not restarting")
			}
		}
	}()

	w.logger.Info().
		Str("task_queue", taskQueue).
		Msg("starting worker")

	if err := wrk.Run(worker.InterruptCh()); err != nil {
		w.logger.Error().
			Err(err).
			Str("task_queue", taskQueue).
			Msg("worker stopped with error")
	}
}

// Stop gracefully stops all workers
func (w *Worker) Stop() error {
	w.logger.Info().Msg("stopping temporal workers")

	w.isRunning.Store(false)

	w.mu.Lock()
	defer w.mu.Unlock()

	for taskQueue, wrk := range w.workers {
		wrk.Stop()
		w.logger.Info().
			Str("task_queue", taskQueue).
			Msg("stopped worker")
	}

	w.logger.Info().Msg("all temporal workers stopped")
	return nil
}

// IsHealthy returns true if the worker is running and healthy
func (w *Worker) IsHealthy() bool {
	return w.isRunning.Load() && w.panicCount.Load() < 10
}

// GetStats returns basic statistics about the worker
func (w *Worker) GetStats() WorkflowStats {
	uptime := ""
	if w.isRunning.Load() && !w.startTime.IsZero() {
		uptime = time.Since(w.startTime).String()
	}

	var lastPanic *time.Time
	if p := w.lastPanic.Load(); p != nil {
		lastPanic = p
	}

	return WorkflowStats{
		IsRunning:  w.isRunning.Load(),
		StartTime:  w.startTime,
		Uptime:     uptime,
		PanicCount: int(w.panicCount.Load()),
		LastPanic:  lastPanic,
	}
}
