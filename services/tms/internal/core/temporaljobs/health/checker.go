package health

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/zap"
)

type WorkerHealthStatus struct {
	WorkerName       string    `json:"workerName"`
	TaskQueue        string    `json:"taskQueue"`
	IsHealthy        bool      `json:"isHealthy"`
	RegistrationTime time.Time `json:"registrationTime"`
	ErrorMessage     string    `json:"errorMessage,omitempty"`
}

type HealthProbe interface {
	CheckHealth(ctx context.Context) []WorkerHealthStatus
	IsReady() bool
}

type WorkerRegistrar interface {
	RegisterWorker(name, taskQueue string)
}

type Checker struct {
	workerManager *registry.WorkerManager
	workers       map[string]*workerInfo
	mu            sync.RWMutex
	logger        *zap.Logger
}

var (
	_ HealthProbe     = (*Checker)(nil)
	_ WorkerRegistrar = (*Checker)(nil)
)

type workerInfo struct {
	name             string
	taskQueue        string
	registrationTime time.Time
}

func NewChecker(manager *registry.WorkerManager, logger *zap.Logger) *Checker {
	return &Checker{
		workerManager: manager,
		workers:       make(map[string]*workerInfo),
		logger:        logger.Named("worker-health-checker"),
	}
}

func (c *Checker) RegisterWorker(name, taskQueue string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.workers[name] = &workerInfo{
		name:             name,
		taskQueue:        taskQueue,
		registrationTime: time.Now(),
	}

	c.logger.Debug("worker registered for health tracking",
		zap.String("name", name),
		zap.String("taskQueue", taskQueue),
	)
}

func (c *Checker) CheckHealth(_ context.Context) []WorkerHealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := make([]WorkerHealthStatus, 0, len(c.workers))

	for name, info := range c.workers {
		status := WorkerHealthStatus{
			WorkerName:       name,
			TaskQueue:        info.taskQueue,
			RegistrationTime: info.registrationTime,
			IsHealthy:        true,
		}

		_, exists := c.workerManager.GetWorker(name)
		if !exists {
			status.IsHealthy = false
			status.ErrorMessage = "worker not registered with manager"
		}

		statuses = append(statuses, status)
	}

	return statuses
}

func (c *Checker) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.workers) == 0 {
		return false
	}

	for name := range c.workers {
		if _, exists := c.workerManager.GetWorker(name); !exists {
			return false
		}
	}

	return true
}
