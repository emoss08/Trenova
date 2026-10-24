package scriptrunner

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"go.uber.org/fx"
)

var Module = fx.Module("script-runner",
	fx.Provide(
		New,
		func(svc *Service) scriptports.Runner { return svc },
	),
)

var WarmModule = fx.Module("script-runner-warmup",
	fx.Invoke(func(svc *Service, lc fx.Lifecycle) {
		lc.Append(fx.StartHook(func() { go svc.Warm() }))
	}),
)
