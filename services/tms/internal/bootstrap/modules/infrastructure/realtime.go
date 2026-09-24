package infrastructure

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/realtimeservice"
	"github.com/emoss08/trenova/internal/infrastructure/realtimebroker"
	"go.uber.org/fx"
)

// RealtimePublisherModule lets a process write to the realtime bus. Every
// process that changes records needs it, the worker included.
var RealtimePublisherModule = fx.Module("realtime-publisher",
	fx.Provide(
		fx.Annotate(
			realtimebroker.NewPublisher,
			fx.As(fx.Self()),
			fx.As(new(services.RealtimePublisher)),
		),
	),
)

// RealtimeGatewayModule serves event streams. Only the API process holds
// readers' connections, so only it reads the bus.
var RealtimeGatewayModule = fx.Module("realtime-gateway",
	fx.Provide(
		fx.Annotate(
			realtimebroker.NewBroker,
			fx.As(new(services.RealtimeBroker)),
		),
		realtimeservice.NewGateway,
	),
)
