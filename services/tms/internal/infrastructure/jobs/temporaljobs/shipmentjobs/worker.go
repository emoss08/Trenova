/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/temporalutils"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
)

type WorkerParams struct {
	fx.In

	LC         fx.Lifecycle
	Client     client.Client
	Logger     *logger.Logger
	Config     *config.Manager
	Activities *Activities
}

func NewWorker(p WorkerParams) error {
	log := p.Logger.With().
		Str("component", "shipment-worker").
		Logger()

	workerConfig := p.Config.Temporal().Workers.Shipment
	workerOptions := temporalutils.BuildWorkerOptions(workerConfig)

	w := worker.New(p.Client,
		temporaltype.ShipmentTaskQueue,
		workerOptions)

	for _, wf := range RegisterWorkflows() {
		w.RegisterWorkflowWithOptions(wf.Fn, workflow.RegisterOptions{
			Name: wf.Name,
		})
	}

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
