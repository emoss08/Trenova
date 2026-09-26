package aitrainingservice

import (
	"go.uber.org/fx"
)

var Module = fx.Module("ai-training-service",
	fx.Provide(
		NewOperator,
		AsOperator,
		NewRunner,
		AsRunner,
		NewHistory,
		NewRenderer,
		AsRenderer,
	),
)
