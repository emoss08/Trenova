/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"context"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// WorkflowRegistry manages workflow and activity registrations
type WorkflowRegistry interface {
	RegisterWorkflow(name string, fn any)
	RegisterActivity(name string, fn any)
	GetWorkflows() map[string]any
	GetActivities() map[string]any
}

// WorkerManager manages Temporal workers
type WorkerManager interface {
	Start() error
	Stop() error
	RegisterWorkflow(workflowFunc any)
	RegisterActivity(activityFunc any)
	RegisterWorkflowWithOptions(workflowFunc any, options workflow.RegisterOptions)
	IsHealthy() bool
	GetStats() WorkflowStats
}

// ScheduleManager manages Temporal schedules
type ScheduleManager interface {
	Start() error
	Stop() error
	CreateSchedule(ctx context.Context, id string, spec client.ScheduleSpec, action client.ScheduleAction) error
	DeleteSchedule(ctx context.Context, id string) error
	PauseSchedule(ctx context.Context, id string) error
	UnpauseSchedule(ctx context.Context, id string) error
	GetSchedule(ctx context.Context, id string) (client.ScheduleHandle, error)
}

// WorkflowService provides workflow execution capabilities
type WorkflowService interface {
	ExecuteWorkflow(ctx context.Context, workflowName string, payload any, opts *WorkflowOptions) (client.WorkflowRun, error)
	ExecuteWorkflowWithDelay(ctx context.Context, workflowName string, payload any, delay time.Duration, opts *WorkflowOptions) (client.WorkflowRun, error)
	CancelWorkflow(ctx context.Context, workflowID string, runID string) error
	GetWorkflowInfo(ctx context.Context, workflowID string, runID string) (client.WorkflowRun, error)
	GetStats() WorkflowStats
}

// WorkflowDefinition defines a workflow with its configuration
type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}

// ActivityDefinition defines an activity with its configuration
type ActivityDefinition struct {
	Name        string
	Fn          any
	Description string
}

// ScheduleDefinition defines a schedule configuration
type ScheduleDefinition struct {
	ID          string
	Description string
	CronSpec    string
	Workflow    string
	TaskQueue   string
	Payload     any
}

// WorkerConfig defines worker configuration
type WorkerConfig struct {
	TaskQueue                        string
	MaxConcurrentActivityExecutions  int
	MaxConcurrentWorkflowExecutions  int
	MaxConcurrentLocalActivityExecutions int
	WorkerOptions                   worker.Options
}