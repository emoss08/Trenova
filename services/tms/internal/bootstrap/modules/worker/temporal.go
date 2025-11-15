package worker

import (
	"context"

	"github.com/emoss08/trenova/internal/core/temporaljobs/auditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/emailjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/notificationjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reportjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/searchjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/workflowjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemporalWorkerParams struct {
	fx.In

	Client               client.Client
	WorkerManager        *registry.WorkerManager
	AuditRegistry        *auditjobs.Registry
	NotificationRegistry *notificationjobs.Registry
	EmailRegistry        *emailjobs.Registry
	ReportRegistry       *reportjobs.Registry
	SearchRegistry       *searchjobs.Registry
	ShipmentRegistry     *shipmentjobs.Registry
	WorkflowRegistry     *workflowjobs.Registry
	Config               *config.Config
	Logger               *zap.Logger
	LC                   fx.Lifecycle
}

//nolint:gocritic // This is a constructor
func NewTemporalWorkers(p TemporalWorkerParams) error {
	log := p.Logger.Named("temporal-workers")

	// Register all workers with the manager
	if err := p.WorkerManager.Register(p.AuditRegistry); err != nil {
		log.Error("failed to register audit worker", zap.Error(err))
		return err
	}

	if err := p.WorkerManager.Register(p.NotificationRegistry); err != nil {
		log.Error("failed to register notification worker", zap.Error(err))
		return err
	}

	if err := p.WorkerManager.Register(p.EmailRegistry); err != nil {
		log.Error("failed to register email worker", zap.Error(err))
		return err
	}

	if err := p.WorkerManager.Register(p.ReportRegistry); err != nil {
		log.Error("failed to register report worker", zap.Error(err))
		return err
	}

	if err := p.WorkerManager.Register(p.SearchRegistry); err != nil {
		log.Error("failed to register search worker", zap.Error(err))
		return err
	}

	if err := p.WorkerManager.Register(p.ShipmentRegistry); err != nil {
		log.Error("failed to register shipment worker", zap.Error(err))
		return err
	}

	if err := p.WorkerManager.Register(p.WorkflowRegistry); err != nil {
		log.Error("failed to register workflow worker", zap.Error(err))
		return err
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("starting temporal workers")
			return p.WorkerManager.StartAll(ctx)
		},
		OnStop: func(ctx context.Context) error {
			log.Info("stopping temporal workers")
			return p.WorkerManager.StopAll(ctx)
		},
	})

	return nil
}
