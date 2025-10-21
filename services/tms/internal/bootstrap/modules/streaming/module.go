package streaming

import (
	"github.com/emoss08/trenova/internal/infrastructure/streaming"
	"go.uber.org/fx"
)

var Module = fx.Module("streaming", fx.Provide(streaming.NewService))
