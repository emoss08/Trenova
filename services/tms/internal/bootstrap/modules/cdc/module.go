package cdc

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/cdc"
	"github.com/emoss08/trenova/internal/infrastructure/cdc/handlers"
	"go.uber.org/fx"
)

var Module = fx.Module("cdc-service",
	fx.Provide(
		fx.Annotate(
			cdc.NewKafkaConsumer,
			fx.As(new(services.CDCService)),
		),
		handlers.NewShipmentCDCHandler,
	),
	fx.Invoke(registerCDCHandlers),
)

func registerCDCHandlers(
	lifecycle fx.Lifecycle,
	cdcService services.CDCService,
	shipmentHandler services.CDCEventHandler,
) {
	cdcService.RegisterHandler("shipments", shipmentHandler)

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return cdcService.Start()
		},
		OnStop: func(context.Context) error {
			return cdcService.Stop()
		},
	})
}
