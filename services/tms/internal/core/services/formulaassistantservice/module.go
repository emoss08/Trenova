package formulaassistantservice

import "go.uber.org/fx"

var Module = fx.Module("formulaassistantservice",
	fx.Provide(New),
)
