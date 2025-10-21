package jobscheduler

import (
	"fmt"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

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

func NewScheduleBuilder(scheduleID string) *ScheduleBuilder {
	return &ScheduleBuilder{
		id:            scheduleID,
		overlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		memo:          make(map[string]any),
		spec: client.ScheduleSpec{
			TimeZoneName: "UTC",
		},
	}
}

func (b *ScheduleBuilder) WithDescription(description string) *ScheduleBuilder {
	b.description = description
	b.memo["description"] = description
	return b
}

func (b *ScheduleBuilder) WithInterval(interval time.Duration) *ScheduleBuilder {
	b.spec.Intervals = []client.ScheduleIntervalSpec{
		{Every: interval},
	}
	b.spec.CronExpressions = nil
	return b
}

func (b *ScheduleBuilder) WithCron(expression string) *ScheduleBuilder {
	b.spec.CronExpressions = []string{expression}
	b.spec.Intervals = nil
	return b
}

func (b *ScheduleBuilder) WithWorkflow(workflowType, taskQueue, idPrefix string) *ScheduleBuilder {
	b.workflowType = workflowType
	b.taskQueue = taskQueue
	b.workflowIDPrefix = idPrefix
	return b
}

func (b *ScheduleBuilder) WithMetadata(key, value string) *ScheduleBuilder {
	if b.memo["metadata"] == nil {
		b.memo["metadata"] = make(map[string]string)
	}
	b.memo["metadata"].(map[string]string)[key] = value //nolint:errcheck // We know the type is map[string]string
	return b
}

func (b *ScheduleBuilder) WithOverlapPolicy(policy enums.ScheduleOverlapPolicy) *ScheduleBuilder {
	b.overlapPolicy = policy
	return b
}

func (b *ScheduleBuilder) WithTimeZone(tz string) *ScheduleBuilder {
	b.spec.TimeZoneName = tz
	return b
}

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

type ScheduleConfig struct {
	ID               string
	Description      string
	Schedule         ScheduleSpec
	WorkflowType     string
	TaskQueue        string
	WorkflowIDPrefix string
	Metadata         map[string]string
}

type ScheduleSpec struct {
	Cron     string
	Interval time.Duration
}

func (s ScheduleSpec) IsInterval() bool {
	return s.Interval > 0
}

func BuildSchedule(config *ScheduleConfig) client.ScheduleOptions {
	builder := NewScheduleBuilder(config.ID).
		WithDescription(config.Description).
		WithWorkflow(config.WorkflowType, config.TaskQueue, config.WorkflowIDPrefix)

	if config.Schedule.IsInterval() {
		builder.WithInterval(config.Schedule.Interval)
	} else {
		builder.WithCron(config.Schedule.Cron)
	}

	for k, v := range config.Metadata {
		builder.WithMetadata(k, v)
	}

	return builder.Build()
}

func CompareScheduleSpecs(existing *client.ScheduleSpec, config *ScheduleSpec) bool {
	if existing == nil {
		return true
	}

	if config.IsInterval() {
		if len(existing.Intervals) == 0 {
			return true
		}
		return existing.Intervals[0].Every != config.Interval
	}

	if len(existing.CronExpressions) == 0 {
		return true
	}
	return existing.CronExpressions[0] != config.Cron
}

func UpdateScheduleSpec(schedule *client.ScheduleUpdateInput, config *ScheduleSpec) {
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
