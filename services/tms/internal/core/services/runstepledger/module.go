package runstepledger

import "go.uber.org/fx"

var Module = fx.Module("runstepledger",
	fx.Provide(New),
)
