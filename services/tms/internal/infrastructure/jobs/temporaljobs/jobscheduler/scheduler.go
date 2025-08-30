package jobscheduler

import (
	"context"
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
	if err := s.scheduleCancelShipmentsByCreatedAt(ctx); err != nil {
		return err
	}

	if err := s.scheduleDeleteAuditEntries(ctx); err != nil {
		return err
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

func (s *Scheduler) scheduleCancelShipmentsByCreatedAt(ctx context.Context) error {
	scheduleID := temporaltype.CancelShipmentsScheduleID

	// * Check if schedule already exists
	scheduleClient := s.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err == nil {
		s.l.Info().
			Str("scheduleID", scheduleID).
			Msg("schedule already exists, skipping creation")
		s.schedules[scheduleID] = handle
		return nil
	}

	scheduleOptions := client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"0 0 * * *"}, // * every day at midnight
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "cancel-shipments-scheduled-" + time.Now().Format("20060102"),
			Workflow:  "CancelShipmentsByCreatedAtWorkflow",
			TaskQueue: temporaltype.ShipmentTaskQueue,
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	newHandle, err := s.client.ScheduleClient().Create(ctx, scheduleOptions)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to create schedule")
		return err
	}

	s.schedules[scheduleID] = newHandle
	s.l.Info().
		Str("scheduleID", newHandle.GetID()).
		Msg("schedule created")

	return nil
}

func (s *Scheduler) scheduleDeleteAuditEntries(ctx context.Context) error {
	scheduleID := temporaltype.DeleteAuditEntriesScheduleID

	// * Check if schedule already exists
	scheduleClient := s.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err == nil {
		s.l.Info().
			Str("scheduleID", scheduleID).
			Msg("schedule already exists, skipping creation")
		s.schedules[scheduleID] = handle
		return nil
	}

	scheduleOptions := client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"0 0 * * *"}, // * every day at midnight
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "delete-audit-entries-scheduled-" + time.Now().Format("20060102"),
			Workflow:  "DeleteAuditEntriesWorkflow",
			TaskQueue: temporaltype.SystemTaskQueue,
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	newHandle, err := s.client.ScheduleClient().Create(ctx, scheduleOptions)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to create schedule")
		return err
	}

	s.schedules[scheduleID] = newHandle
	s.l.Info().
		Str("scheduleID", newHandle.GetID()).
		Msg("schedule created")

	return nil
}
