/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

// SchedulerParams defines dependencies for the cron scheduler
type SchedulerParams struct {
	fx.In

	Logger *logger.Logger
	Client client.Client
	Config *config.Manager
}

// CronScheduler manages recurring workflow schedules using Temporal's schedule feature
type CronScheduler struct {
	client    client.Client
	logger    *zerolog.Logger
	config    *config.Manager
	schedules map[string]client.ScheduleHandle
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(p SchedulerParams) *CronScheduler {
	log := p.Logger.With().
		Str("component", "temporal-cron-scheduler").
		Logger()

	return &CronScheduler{
		client:    p.Client,
		logger:    &log,
		config:    p.Config,
		schedules: make(map[string]client.ScheduleHandle),
	}
}

// Start initializes and starts all scheduled workflows
func (cs *CronScheduler) Start() error {
	cs.logger.Info().Msg("starting temporal cron scheduler")

	ctx := context.Background()

	// Schedule pattern analysis - every minute for testing (normally daily at 2 AM)
	if err := cs.schedulePatternAnalysis(ctx); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule pattern analysis")
		// Don't return error, continue with other schedules
	}

	// Schedule suggestion expiration - every 2 minutes for testing (normally every 6 hours)
	if err := cs.scheduleExpireSuggestions(ctx); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule expire suggestions")
	}

	// Schedule compliance checks - every 3 minutes for testing (normally daily at 3 AM)
	if err := cs.scheduleComplianceChecks(ctx); err != nil {
		cs.logger.Error().Err(err).Msg("failed to schedule compliance checks")
	}

	cs.logger.Info().
		Int("scheduled_count", len(cs.schedules)).
		Msg("temporal cron scheduler started")

	return nil
}

// Stop cancels all scheduled workflows
func (cs *CronScheduler) Stop() error {
	cs.logger.Info().Msg("stopping temporal cron scheduler")

	ctx := context.Background()
	for id, handle := range cs.schedules {
		if err := handle.Delete(ctx); err != nil {
			cs.logger.Error().
				Err(err).
				Str("schedule_id", id).
				Msg("failed to delete schedule")
		} else {
			cs.logger.Info().
				Str("schedule_id", id).
				Msg("deleted schedule")
		}
	}

	cs.schedules = make(map[string]client.ScheduleHandle)
	cs.logger.Info().Msg("temporal cron scheduler stopped")
	return nil
}

// schedulePatternAnalysis creates a daily pattern analysis schedule
func (cs *CronScheduler) schedulePatternAnalysis(ctx context.Context) error {
	scheduleID := "pattern-analysis-daily"

	// Check if schedule already exists
	scheduleClient := cs.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err == nil {
		cs.logger.Info().
			Str("schedule_id", scheduleID).
			Msg("schedule already exists, skipping creation")
		cs.schedules[scheduleID] = handle
		return nil
	}

	// Create payload for the workflow
	// Create a generic payload map for the workflow
	payload := map[string]any{
		"basePayload": map[string]any{
			"jobId":          pulid.MustNew("job_").String(),
			"organizationId": pulid.MustNew("org_").String(),
			"businessUnitId": pulid.MustNew("bu_").String(),
			"timestamp":      timeutils.NowUnix(),
			"metadata": map[string]any{
				"scheduled": true,
				"type":      "daily_analysis",
			},
		},
		"minFrequency":  3,
		"triggerReason": "scheduled",
	}

	// Create schedule spec for every minute (for testing)
	spec := client.ScheduleSpec{
		CronExpressions: []string{"* * * * *"}, // Every minute for testing
	}

	// Create schedule options
	scheduleOptions := client.ScheduleOptions{
		ID:   scheduleID,
		Spec: spec,
		Action: &client.ScheduleWorkflowAction{
			ID:        "pattern-analysis-scheduled-" + time.Now().Format("20060102"),
			TaskQueue: "pattern-analysis-tasks",
			Workflow:  "PatternAnalysisWorkflow",
			Args:      []any{payload},
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	newHandle, err := cs.client.ScheduleClient().Create(ctx, scheduleOptions)
	if err != nil {
		return fmt.Errorf("create pattern analysis schedule: %w", err)
	}
	handle = newHandle

	cs.schedules[scheduleID] = handle
	cs.logger.Info().
		Str("schedule_id", scheduleID).
		Str("cron", spec.CronExpressions[0]).
		Msg("scheduled pattern analysis")

	return nil
}

// scheduleExpireSuggestions creates a recurring schedule to expire old suggestions
func (cs *CronScheduler) scheduleExpireSuggestions(ctx context.Context) error {
	scheduleID := "expire-suggestions-recurring"

	// Check if schedule already exists
	scheduleClient := cs.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err == nil {
		cs.logger.Info().
			Str("schedule_id", scheduleID).
			Msg("schedule already exists, skipping creation")
		cs.schedules[scheduleID] = handle
		return nil
	}

	payload := map[string]any{
		"basePayload": map[string]any{
			"jobId":          pulid.MustNew("job_").String(),
			"organizationId": pulid.MustNew("org_").String(),
			"businessUnitId": pulid.MustNew("bu_").String(),
			"timestamp":      timeutils.NowUnix(),
			"metadata": map[string]any{
				"scheduled": true,
				"type":      "expire_suggestions",
			},
		},
		"batchSize": 100,
	}

	spec := client.ScheduleSpec{
		CronExpressions: []string{"*/2 * * * *"}, // Every 2 minutes for testing
	}

	scheduleOptions := client.ScheduleOptions{
		ID:   scheduleID,
		Spec: spec,
		Action: &client.ScheduleWorkflowAction{
			ID:        "expire-suggestions-" + time.Now().Format("20060102-150405"),
			TaskQueue: "default-tasks",
			Workflow:  "ExpireSuggestionsWorkflow",
			Args:      []any{payload},
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	newHandle, err := cs.client.ScheduleClient().Create(ctx, scheduleOptions)
	if err != nil {
		return fmt.Errorf("create expire suggestions schedule: %w", err)
	}
	handle = newHandle

	cs.schedules[scheduleID] = handle
	cs.logger.Info().
		Str("schedule_id", scheduleID).
		Str("cron", spec.CronExpressions[0]).
		Msg("scheduled expire suggestions")

	return nil
}

// scheduleComplianceChecks creates a daily compliance check schedule
func (cs *CronScheduler) scheduleComplianceChecks(ctx context.Context) error {
	scheduleID := "compliance-checks-daily"

	// Check if schedule already exists
	scheduleClient := cs.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err == nil {
		cs.logger.Info().
			Str("schedule_id", scheduleID).
			Msg("schedule already exists, skipping creation")
		cs.schedules[scheduleID] = handle
		return nil
	}

	payload := map[string]any{
		"basePayload": map[string]any{
			"jobId":          pulid.MustNew("job_").String(),
			"organizationId": pulid.MustNew("org_").String(),
			"businessUnitId": pulid.MustNew("bu_").String(),
			"timestamp":      timeutils.NowUnix(),
			"metadata": map[string]any{
				"scheduled": true,
				"type":      "daily_compliance",
			},
		},
		"checkType": "all",
	}

	spec := client.ScheduleSpec{
		CronExpressions: []string{"*/3 * * * *"}, // Every 3 minutes for testing
	}

	scheduleOptions := client.ScheduleOptions{
		ID:   scheduleID,
		Spec: spec,
		Action: &client.ScheduleWorkflowAction{
			ID:        "compliance-check-" + time.Now().Format("20060102"),
			TaskQueue: "compliance-tasks",
			Workflow:  "ComplianceCheckWorkflow",
			Args:      []any{payload},
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	newHandle, err := cs.client.ScheduleClient().Create(ctx, scheduleOptions)
	if err != nil {
		return fmt.Errorf("create compliance checks schedule: %w", err)
	}
	handle = newHandle

	cs.schedules[scheduleID] = handle
	cs.logger.Info().
		Str("schedule_id", scheduleID).
		Str("cron", spec.CronExpressions[0]).
		Msg("scheduled compliance checks")

	return nil
}

// CreateCustomSchedule allows creating custom scheduled workflows
func (cs *CronScheduler) CreateCustomSchedule(
	ctx context.Context,
	scheduleID string,
	cronExpression string,
	workflowName string,
	payload any,
	taskQueue string,
) error {
	spec := client.ScheduleSpec{
		CronExpressions: []string{cronExpression},
	}

	scheduleOptions := client.ScheduleOptions{
		ID:   scheduleID,
		Spec: spec,
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-%s", workflowName, time.Now().Format("20060102-150405")),
			TaskQueue: taskQueue,
			Workflow:  workflowName,
			Args:      []any{payload},
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	handle, err := cs.client.ScheduleClient().Create(ctx, scheduleOptions)
	if err != nil {
		return fmt.Errorf("create custom schedule: %w", err)
	}

	cs.schedules[scheduleID] = handle
	cs.logger.Info().
		Str("schedule_id", scheduleID).
		Str("cron", cronExpression).
		Str("workflow", workflowName).
		Msg("created custom schedule")

	return nil
}

// RemoveSchedule removes a scheduled workflow
func (cs *CronScheduler) RemoveSchedule(ctx context.Context, scheduleID string) error {
	handle, exists := cs.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	if err := handle.Delete(ctx); err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}

	delete(cs.schedules, scheduleID)
	cs.logger.Info().
		Str("schedule_id", scheduleID).
		Msg("removed schedule")

	return nil
}
