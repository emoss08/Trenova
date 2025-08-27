package shipmentjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
)

type ShipmentWorkerParams struct {
	fx.In

	LC         fx.Lifecycle
	Client     client.Client
	Logger     *logger.Logger
	Activities *ShipmentJobsActivities
}

func NewShipmentWorker(p ShipmentWorkerParams) error {
	log := p.Logger.With().
		Str("component", "temporal-worker").
		Logger()

	w := worker.New(p.Client,
		temporaltype.TaskQueueShipmentWorker,
		worker.Options{
			EnableSessionWorker: true,
		})

	w.RegisterWorkflow(DuplicateShipmentWorkflow)
	w.RegisterActivity(p.Activities)

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := w.Run(worker.InterruptCh()); err != nil {
					log.Error().Err(err).Msg("failed to run shipment worker")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			w.Stop()
			p.Client.Close()
			return nil
		},
	})

	return nil
}
