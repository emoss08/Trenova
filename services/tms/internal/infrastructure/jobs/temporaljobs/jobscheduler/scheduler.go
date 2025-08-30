package jobscheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"github.com/rs/zerolog"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type SchedulerParams struct {
	fx.In

	Client client.Client
	Logger *logger.Logger
	LC     fx.Lifecycle
}

type Scheduler struct {
	client    client.Client
	l         *zerolog.Logger
	schedules map[string]client.ScheduleHandle
}

type ScheduleConfig struct {
	ID               string
	Description      string
	CronExpression   string
	WorkflowType     string
	TaskQueue        string
	WorkflowIDPrefix string
	Metadata         map[string]string
}

func NewScheduler(p SchedulerParams) *Scheduler {
	log := p.Logger.With().
		Str("component", "job-scheduler").
		Logger()

	s := &Scheduler{
		client:    p.Client,
		l:         &log,
		schedules: make(map[string]client.ScheduleHandle),
	}

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info().Msg("starting job scheduler")
			return s.Start()
		},
		OnStop: func(context.Context) error {
			log.Info().Msg("stopping job scheduler")
			return s.Stop()
		},
	})

	return s
}

func (s *Scheduler) Start() error {
	ctx := context.Background()

	schedules := []ScheduleConfig{
		{
			ID:               temporaltype.CancelShipmentsScheduleID,
			Description:      "Automatically cancel shipments older than 30 days",
			CronExpression:   "0 0 * * *", // Daily at midnight
			WorkflowType:     "CancelShipmentsByCreatedAtWorkflow",
			TaskQueue:        temporaltype.ShipmentTaskQueue,
			WorkflowIDPrefix: "cancel-shipments-scheduled",
			Metadata: map[string]string{
				"purpose": "cleanup",
				"target":  "shipments",
			},
		},
		{
			ID:               temporaltype.DeleteAuditEntriesScheduleID,
			Description:      "Delete audit log entries older than 120 days",
			CronExpression:   "0 0 * * *", // Daily at midnight
			WorkflowType:     "DeleteAuditEntriesWorkflow",
			TaskQueue:        temporaltype.SystemTaskQueue,
			WorkflowIDPrefix: "delete-audit-entries-scheduled",
			Metadata: map[string]string{
				"purpose": "cleanup",
				"target":  "audit_logs",
			},
		},
	}

	for i := range schedules {
		if err := s.createOrUpdateSchedule(ctx, &schedules[i]); err != nil {
			s.l.Error().
				Str("scheduleID", schedules[i].ID).
				Err(err).
				Msg("failed to create/update schedule")
			return err
		}
	}

	return nil
}

func (s *Scheduler) Stop() error {
	s.l.Info().Msg("stopping job scheduler")
	ctx := context.Background()

	for idx, schedule := range s.schedules {
		if err := schedule.Delete(ctx); err != nil {
			s.l.Error().
				Str("scheduleID", idx).
				Err(err).
				Msg("failed to delete schedule")
		} else {
			s.l.Info().
				Str("scheduleID", idx).
				Msg("schedule deleted")
		}
	}

	s.schedules = make(map[string]client.ScheduleHandle)
	s.l.Info().Msg("job scheduler stopped")

	return nil
}

func (s *Scheduler) createOrUpdateSchedule(ctx context.Context, config *ScheduleConfig) error {
	scheduleClient := s.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, config.ID)

	existing, err := handle.Describe(ctx)
	if err == nil {
		if s.shouldUpdateSchedule(existing, config) {
			s.l.Info().
				Str("scheduleID", config.ID).
				Msg("updating existing schedule")

			err = handle.Update(ctx, client.ScheduleUpdateOptions{
				DoUpdate: func(schedule client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
					schedule.Description.Schedule.Spec.CronExpressions = []string{
						config.CronExpression,
					}
					schedule.Description.Schedule.Spec.TimeZoneName = "UTC"

					return &client.ScheduleUpdate{
						Schedule: &schedule.Description.Schedule,
					}, nil
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update schedule: %w", err)
			}
		} else {
			s.l.Info().
				Str("scheduleID", config.ID).
				Msg("schedule already exists with correct configuration")
		}

		s.schedules[config.ID] = handle
		return nil
	}

	// Create new schedule with memo
	memo := map[string]any{
		"description": config.Description,
		"metadata":    config.Metadata,
		"createdAt":   time.Now().Format(time.RFC3339),
	}

	scheduleOptions := client.ScheduleOptions{
		ID: config.ID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{config.CronExpression},
			TimeZoneName:    "UTC",
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-%s", config.WorkflowIDPrefix, time.Now().Format("20060102")),
			Workflow:  config.WorkflowType,
			TaskQueue: config.TaskQueue,
			Memo:      memo,
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	newHandle, err := scheduleClient.Create(ctx, scheduleOptions)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	s.schedules[config.ID] = newHandle
	s.l.Info().
		Str("scheduleID", newHandle.GetID()).
		Str("description", config.Description).
		Msg("schedule created")

	return nil
}

func (s *Scheduler) shouldUpdateSchedule(
	existing *client.ScheduleDescription,
	config *ScheduleConfig,
) bool {
	if len(existing.Schedule.Spec.CronExpressions) == 0 ||
		existing.Schedule.Spec.CronExpressions[0] != config.CronExpression {
		return true
	}

	if action, ok := existing.Schedule.Action.(*client.ScheduleWorkflowAction); ok {
		if action.Workflow != config.WorkflowType || action.TaskQueue != config.TaskQueue {
			return true
		}
	}

	return false
}

// PauseSchedule pauses a schedule by ID
func (s *Scheduler) PauseSchedule(ctx context.Context, scheduleID string) error {
	handle, exists := s.schedules[scheduleID]
	if !exists {
		handle = s.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}

	err := handle.Pause(ctx, client.SchedulePauseOptions{
		Note: fmt.Sprintf("Paused at %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		return fmt.Errorf("failed to pause schedule %s: %w", scheduleID, err)
	}

	s.l.Info().
		Str("scheduleID", scheduleID).
		Msg("schedule paused")

	return nil
}

// UnpauseSchedule resumes a paused schedule
func (s *Scheduler) UnpauseSchedule(ctx context.Context, scheduleID string) error {
	handle, exists := s.schedules[scheduleID]
	if !exists {
		handle = s.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}

	err := handle.Unpause(ctx, client.ScheduleUnpauseOptions{
		Note: fmt.Sprintf("Unpaused at %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		return fmt.Errorf("failed to unpause schedule %s: %w", scheduleID, err)
	}

	s.l.Info().
		Str("scheduleID", scheduleID).
		Msg("schedule unpaused")

	return nil
}

// TriggerSchedule manually triggers a schedule execution
func (s *Scheduler) TriggerSchedule(ctx context.Context, scheduleID string) error {
	handle, exists := s.schedules[scheduleID]
	if !exists {
		handle = s.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}

	err := handle.Trigger(ctx, client.ScheduleTriggerOptions{
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL,
	})
	if err != nil {
		return fmt.Errorf("failed to trigger schedule %s: %w", scheduleID, err)
	}

	s.l.Info().
		Str("scheduleID", scheduleID).
		Msg("schedule triggered manually")

	return nil
}

// UpdateScheduleCron updates the cron expression of a schedule
func (s *Scheduler) UpdateScheduleCron(
	ctx context.Context,
	scheduleID string,
	newCronExpression string,
) error {
	handle, exists := s.schedules[scheduleID]
	if !exists {
		handle = s.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}

	err := handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(schedule client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			schedule.Description.Schedule.Spec.CronExpressions = []string{newCronExpression}

			return &client.ScheduleUpdate{
				Schedule: &schedule.Description.Schedule,
			}, nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update schedule cron for %s: %w", scheduleID, err)
	}

	s.l.Info().
		Str("scheduleID", scheduleID).
		Str("newCron", newCronExpression).
		Msg("schedule cron expression updated")

	return nil
}

// GetScheduleInfo retrieves information about a schedule
func (s *Scheduler) GetScheduleInfo(
	ctx context.Context,
	scheduleID string,
) (*client.ScheduleDescription, error) {
	handle, exists := s.schedules[scheduleID]
	if !exists {
		handle = s.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}

	description, err := handle.Describe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule info for %s: %w", scheduleID, err)
	}

	return description, nil
}

// ListSchedules returns information about all managed schedules
func (s *Scheduler) ListSchedules(ctx context.Context) ([]client.ScheduleListEntry, error) {
	var schedules []client.ScheduleListEntry

	iter, err := s.client.ScheduleClient().List(ctx, client.ScheduleListOptions{
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule iterator: %w", err)
	}

	for iter.HasNext() {
		schedule, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to list schedules: %w", err)
		}
		schedules = append(schedules, *schedule)
	}

	return schedules, nil
}
