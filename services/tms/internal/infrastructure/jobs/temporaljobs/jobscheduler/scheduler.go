package jobscheduler

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/temporalutils"
	"github.com/rs/zerolog"
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
	manager *temporalutils.ScheduleManager
	l       *zerolog.Logger
}

func NewScheduler(p SchedulerParams) *Scheduler {
	log := p.Logger.With().
		Str("component", "job-scheduler").
		Logger()

	s := &Scheduler{
		manager: temporalutils.NewScheduleManager(p.Client, &log),
		l:       &log,
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

	schedules := []temporalutils.ScheduleConfig{
		{
			ID:               temporaltype.CancelShipmentsScheduleID,
			Description:      "Automatically cancel shipments older than 30 days",
			Schedule:         temporalutils.ScheduleSpec{Cron: "0 0 * * *"}, // Daily at midnight
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
			Schedule:         temporalutils.ScheduleSpec{Cron: "0 0 * * *"}, // Daily at midnight
			WorkflowType:     "DeleteAuditEntriesWorkflow",
			TaskQueue:        temporaltype.SystemTaskQueue,
			WorkflowIDPrefix: "delete-audit-entries-scheduled",
			Metadata: map[string]string{
				"purpose": "cleanup",
				"target":  "audit_logs",
			},
		},
		{
			ID:               "flush-audit-buffer-schedule",
			Description:      "Flush audit buffer entries for batch processing",
			Schedule:         temporalutils.ScheduleSpec{Interval: 30 * time.Second}, // Every 30 seconds
			WorkflowType:     "ScheduledAuditFlushWorkflow",
			TaskQueue:        "audit-queue",
			WorkflowIDPrefix: "flush-audit-buffer-scheduled",
			Metadata: map[string]string{
				"purpose": "batch-processing",
				"target":  "audit_buffer",
			},
		},
	}

	for _, schedule := range schedules {
		if err := s.manager.CreateOrUpdateSchedule(ctx, schedule); err != nil {
			s.l.Error().
				Str("scheduleID", schedule.ID).
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

	if err := s.manager.Clear(ctx); err != nil {
		s.l.Error().Err(err).Msg("error clearing schedules")
		return err
	}

	s.l.Info().Msg("job scheduler stopped")
	return nil
}