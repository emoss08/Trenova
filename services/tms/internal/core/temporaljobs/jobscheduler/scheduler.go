package jobscheduler

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Client  client.Client
	Logger  *zap.Logger
	LC      fx.Lifecycle
	Manager *Manager
}

type Scheduler struct {
	manager *Manager
	l       *zap.Logger
}

func NewScheduler(p Params) *Scheduler {
	log := p.Logger.With(zap.String("component", "job-scheduler"))

	s := &Scheduler{
		manager: p.Manager,
		l:       log,
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("starting job scheduler")
			return s.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			log.Info("stopping job scheduler")
			return s.Stop(ctx)
		},
	})

	return s
}

func (s *Scheduler) Start(ctx context.Context) error {
	log := s.l.With(zap.String("operation", "Start"))
	schedules := []ScheduleConfig{
		{
			ID:               temporaltype.CancelShipmentsScheduleID,
			Description:      "Automatically cancel shipments older than 30 days",
			Schedule:         ScheduleSpec{Cron: "0 0 * * *"}, // Daily at midnight
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
			Schedule:         ScheduleSpec{Cron: "0 0 * * *"}, // Daily at midnight
			WorkflowType:     "DeleteAuditEntriesWorkflow",
			TaskQueue:        temporaltype.SystemTaskQueue,
			WorkflowIDPrefix: "delete-audit-entries-scheduled",
			Metadata: map[string]string{
				"purpose": "cleanup",
				"target":  "audit_logs",
			},
		},
		{
			ID:          "flush-audit-buffer-schedule",
			Description: "Flush audit buffer entries for batch processing",
			Schedule: ScheduleSpec{
				Interval: 30 * time.Minute, // Every 30 minutes
			},
			WorkflowType:     "ScheduledAuditFlushWorkflow",
			TaskQueue:        temporaltype.AuditTaskQueue,
			WorkflowIDPrefix: "flush-audit-buffer-scheduled",
			Metadata: map[string]string{
				"purpose": "batch-processing",
				"target":  "audit_buffer",
			},
		},
	}

	for _, sched := range schedules {
		if err := s.manager.CreateOrUpdateSchedule(ctx, &sched); err != nil {
			log.Error("failed to create/update schedule", zap.Error(err))
			return err
		}
	}

	return nil
}

func (s *Scheduler) Stop(ctx context.Context) error {
	log := s.l.With(zap.String("operation", "Stop"))

	if err := s.manager.Clear(ctx); err != nil {
		log.Error("failed to clear schedules", zap.Error(err))
		return err
	}

	return nil
}
