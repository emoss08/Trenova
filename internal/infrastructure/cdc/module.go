package cdc

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/cdc/handlers"
	"go.uber.org/fx"
)

// Module provides the CDC service for dependency injection
var Module = fx.Module("cdc-service",
	fx.Provide(
		fx.Annotate(
			NewKafkaConsumerService,
			fx.As(new(services.CDCService)),
		),
		handlers.NewShipmentCDCHandler,
	),
	fx.Invoke(registerCDCHandlers),
)

// registerCDCHandlers registers all CDC handlers and starts the CDC service
func registerCDCHandlers(
	lifecycle fx.Lifecycle,
	cdcService services.CDCService,
	shipmentHandler services.CDCEventHandler,
) {
	// Register the shipment handler
	cdcService.RegisterHandler("shipments", shipmentHandler)

	// Add lifecycle hooks to start/stop the CDC service
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return cdcService.Start()
		},
		OnStop: func(context.Context) error {
			return cdcService.Stop()
		},
	})
}
