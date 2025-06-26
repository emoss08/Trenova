package streaming

import (
	"go.uber.org/fx"
)

// Module provides the streaming service for dependency injection
var Module = fx.Module("streaming-service", fx.Provide(
	NewService,
))
