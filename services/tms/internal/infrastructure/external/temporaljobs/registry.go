/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
)

// RegistryParams defines dependencies for the registry
type RegistryParams struct {
	fx.In

	Logger *logger.Logger
}

// Registry implements WorkflowRegistry for managing workflow and activity registrations
type Registry struct {
	mu         sync.RWMutex
	workflows  map[string]*WorkflowDefinition
	activities map[string]*ActivityDefinition
	schedules  map[string]*ScheduleDefinition
	logger     *zerolog.Logger
}

// NewRegistry creates a new workflow registry
func NewRegistry(p RegistryParams) *Registry {
	log := p.Logger.With().
		Str("component", "temporal-registry").
		Logger()

	return &Registry{
		workflows:  make(map[string]*WorkflowDefinition),
		activities: make(map[string]*ActivityDefinition),
		schedules:  make(map[string]*ScheduleDefinition),
		logger:     &log,
	}
}

// RegisterWorkflow registers a workflow definition
func (r *Registry) RegisterWorkflow(def *WorkflowDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def == nil {
		return fmt.Errorf("workflow definition cannot be nil")
	}

	if def.Name == "" {
		return fmt.Errorf("workflow name cannot be empty")
	}

	if def.Fn == nil {
		return fmt.Errorf("workflow function cannot be nil")
	}

	if _, exists := r.workflows[def.Name]; exists {
		return fmt.Errorf("workflow %s already registered", def.Name)
	}

	r.workflows[def.Name] = def
	r.logger.Debug().
		Str("workflow", def.Name).
		Str("task_queue", def.TaskQueue).
		Msg("registered workflow")

	return nil
}

// RegisterActivity registers an activity definition
func (r *Registry) RegisterActivity(def *ActivityDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def == nil {
		return fmt.Errorf("activity definition cannot be nil")
	}

	if def.Name == "" {
		return fmt.Errorf("activity name cannot be empty")
	}

	if def.Fn == nil {
		return fmt.Errorf("activity function cannot be nil")
	}

	if _, exists := r.activities[def.Name]; exists {
		return fmt.Errorf("activity %s already registered", def.Name)
	}

	r.activities[def.Name] = def
	r.logger.Debug().
		Str("activity", def.Name).
		Msg("registered activity")

	return nil
}

// RegisterSchedule registers a schedule definition
func (r *Registry) RegisterSchedule(def *ScheduleDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def == nil {
		return fmt.Errorf("schedule definition cannot be nil")
	}

	if def.ID == "" {
		return fmt.Errorf("schedule ID cannot be empty")
	}

	if _, exists := r.schedules[def.ID]; exists {
		return fmt.Errorf("schedule %s already registered", def.ID)
	}

	r.schedules[def.ID] = def
	r.logger.Debug().
		Str("schedule_id", def.ID).
		Str("workflow", def.Workflow).
		Str("cron", def.CronSpec).
		Msg("registered schedule")

	return nil
}

// GetWorkflow returns a workflow definition by name
func (r *Registry) GetWorkflow(name string) (*WorkflowDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.workflows[name]
	return def, exists
}

// GetActivity returns an activity definition by name
func (r *Registry) GetActivity(name string) (*ActivityDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.activities[name]
	return def, exists
}

// GetSchedule returns a schedule definition by ID
func (r *Registry) GetSchedule(id string) (*ScheduleDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.schedules[id]
	return def, exists
}

// GetAllWorkflows returns all registered workflows
func (r *Registry) GetAllWorkflows() map[string]*WorkflowDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	workflows := make(map[string]*WorkflowDefinition)
	for k, v := range r.workflows {
		workflows[k] = v
	}
	return workflows
}

// GetAllActivities returns all registered activities
func (r *Registry) GetAllActivities() map[string]*ActivityDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	activities := make(map[string]*ActivityDefinition)
	for k, v := range r.activities {
		activities[k] = v
	}
	return activities
}

// GetAllSchedules returns all registered schedules
func (r *Registry) GetAllSchedules() map[string]*ScheduleDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	schedules := make(map[string]*ScheduleDefinition)
	for k, v := range r.schedules {
		schedules[k] = v
	}
	return schedules
}

// ApplyToWorker applies all registered workflows and activities to a worker
func (r *Registry) ApplyToWorker(worker *Worker) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Register all workflows
	for name, def := range r.workflows {
		worker.RegisterWorkflowWithOptions(def.Fn, workflow.RegisterOptions{
			Name: name,
		})
	}

	// Register all activities
	for _, def := range r.activities {
		worker.RegisterActivity(def.Fn)
	}

	r.logger.Info().
		Int("workflows", len(r.workflows)).
		Int("activities", len(r.activities)).
		Msg("applied registrations to worker")

	return nil
}