package streaming

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

// Module provides the streaming service for dependency injection
var Module = fx.Module("streaming-service", fx.Provide(
	ProvideConfig,
	NewService,
))

func ProvideConfig() services.StreamConfig {
	return services.StreamConfig{
		PollInterval:    2000, // 2 seconds
		MaxConnections:  100,  // Max 100 concurrent connections per stream
		StreamTimeout:   1800, // 30 minutes timeout
		EnableHeartbeat: true,
	}
}
