// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package streaming

import (
	"go.uber.org/fx"
)

// Module provides the streaming service for dependency injection
var Module = fx.Module("streaming-service", fx.Provide(
	NewService,
))
