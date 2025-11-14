package storageservice

import "go.uber.org/fx"

var Module = fx.Module(
	"storageservice",
	fx.Provide(NewService),
)
