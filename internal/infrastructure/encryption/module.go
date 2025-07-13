package encryption

import "go.uber.org/fx"

var Module = fx.Module("encryption",
	fx.Provide(NewService),
)
