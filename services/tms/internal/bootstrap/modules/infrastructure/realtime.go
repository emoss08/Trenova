package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/realtimebroker"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/realtimeservice"
	"go.uber.org/fx"
)

// RealtimePublisherModule lets a process write to the realtime bus. Every
// process that changes records needs it, the worker included.
var RealtimePublisherModule = fx.Module("realtime-publisher",
	fx.Provide(
		fx.Annotate(
			realtimebroker.NewPublisher,
			fx.As(fx.Self()),
			fx.As(new(realtimeports.RealtimePublisher)),
		),
		realtimebroker.NewChangeFeed,
	),
)

// RealtimeGatewayModule serves event streams. Only the API process holds
// readers' connections, so only it reads the bus.
var RealtimeGatewayModule = fx.Module("realtime-gateway",
	fx.Provide(
		fx.Annotate(
			realtimebroker.NewBroker,
			fx.As(new(realtimeports.RealtimeBroker)),
		),
		realtimeservice.NewGateway,
	),
)
