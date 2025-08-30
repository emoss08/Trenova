package temporalutils

import (
	"fmt"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// ScheduleBuilder provides a fluent interface for building Temporal schedules
type ScheduleBuilder struct {
	id               string
	description      string
	spec             client.ScheduleSpec
	workflowType     string
	taskQueue        string
	workflowIDPrefix string
	memo             map[string]any
	overlapPolicy    enums.ScheduleOverlapPolicy
}

// NewScheduleBuilder creates a new schedule builder
func NewScheduleBuilder(id string) *ScheduleBuilder {
	return &ScheduleBuilder{
		id:            id,
		overlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		memo:          make(map[string]any),
		spec: client.ScheduleSpec{
			TimeZoneName: "UTC",
		},
	}
}

// WithDescription sets the schedule description
func (b *ScheduleBuilder) WithDescription(description string) *ScheduleBuilder {
	b.description = description
	b.memo["description"] = description
	return b
}

// WithInterval sets an interval-based schedule (for sub-minute frequencies)
func (b *ScheduleBuilder) WithInterval(interval time.Duration) *ScheduleBuilder {
	b.spec.Intervals = []client.ScheduleIntervalSpec{
		{Every: interval},
	}
	b.spec.CronExpressions = nil
	return b
}

// WithCron sets a cron-based schedule
func (b *ScheduleBuilder) WithCron(expression string) *ScheduleBuilder {
	b.spec.CronExpressions = []string{expression}
	b.spec.Intervals = nil
	return b
}

// WithWorkflow sets the workflow configuration
func (b *ScheduleBuilder) WithWorkflow(workflowType, taskQueue, idPrefix string) *ScheduleBuilder {
	b.workflowType = workflowType
	b.taskQueue = taskQueue
	b.workflowIDPrefix = idPrefix
	return b
}

// WithMetadata adds metadata to the schedule
func (b *ScheduleBuilder) WithMetadata(key, value string) *ScheduleBuilder {
	if b.memo["metadata"] == nil {
		b.memo["metadata"] = make(map[string]string)
	}
	b.memo["metadata"].(map[string]string)[key] = value
	return b
}

// WithOverlapPolicy sets the overlap policy
func (b *ScheduleBuilder) WithOverlapPolicy(policy enums.ScheduleOverlapPolicy) *ScheduleBuilder {
	b.overlapPolicy = policy
	return b
}

// WithTimeZone sets the timezone for the schedule
func (b *ScheduleBuilder) WithTimeZone(tz string) *ScheduleBuilder {
	b.spec.TimeZoneName = tz
	return b
}

// Build creates the schedule options
func (b *ScheduleBuilder) Build() client.ScheduleOptions {
	b.memo["createdAt"] = time.Now().Format(time.RFC3339)
	
	return client.ScheduleOptions{
		ID:   b.id,
		Spec: b.spec,
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-%d", b.workflowIDPrefix, time.Now().Unix()),
			Workflow:  b.workflowType,
			TaskQueue: b.taskQueue,
			Memo:      b.memo,
		},
		Overlap: b.overlapPolicy,
	}
}

// ScheduleConfig represents a schedule configuration
type ScheduleConfig struct {
	ID               string
	Description      string
	Schedule         ScheduleSpec
	WorkflowType     string
	TaskQueue        string
	WorkflowIDPrefix string
	Metadata         map[string]string
}

// ScheduleSpec represents either a cron or interval schedule
type ScheduleSpec struct {
	Cron     string
	Interval time.Duration
}

// IsInterval returns true if this is an interval-based schedule
func (s ScheduleSpec) IsInterval() bool {
	return s.Interval > 0
}

// BuildSchedule creates schedule options from config
func BuildSchedule(config ScheduleConfig) client.ScheduleOptions {
	builder := NewScheduleBuilder(config.ID).
		WithDescription(config.Description).
		WithWorkflow(config.WorkflowType, config.TaskQueue, config.WorkflowIDPrefix)
	
	// Add schedule spec
	if config.Schedule.IsInterval() {
		builder.WithInterval(config.Schedule.Interval)
	} else {
		builder.WithCron(config.Schedule.Cron)
	}
	
	// Add metadata
	for k, v := range config.Metadata {
		builder.WithMetadata(k, v)
	}
	
	return builder.Build()
}

// CompareScheduleSpecs checks if two schedule specs are different
func CompareScheduleSpecs(existing *client.ScheduleSpec, config ScheduleSpec) bool {
	if existing == nil {
		return true
	}
	
	if config.IsInterval() {
		// Check if existing uses interval and matches
		if len(existing.Intervals) == 0 {
			return true // Different: existing uses cron, config wants interval
		}
		return existing.Intervals[0].Every != config.Interval
	}
	
	// Config uses cron
	if len(existing.CronExpressions) == 0 {
		return true // Different: existing uses interval, config wants cron
	}
	return existing.CronExpressions[0] != config.Cron
}

// UpdateScheduleSpec updates a schedule's spec during an update operation
func UpdateScheduleSpec(schedule *client.ScheduleUpdateInput, config ScheduleSpec) {
	if config.IsInterval() {
		schedule.Description.Schedule.Spec.Intervals = []client.ScheduleIntervalSpec{
			{Every: config.Interval},
		}
		schedule.Description.Schedule.Spec.CronExpressions = nil
	} else {
		schedule.Description.Schedule.Spec.CronExpressions = []string{config.Cron}
		schedule.Description.Schedule.Spec.Intervals = nil
	}
	schedule.Description.Schedule.Spec.TimeZoneName = "UTC"
}