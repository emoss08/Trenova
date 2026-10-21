package imaging

import "go.uber.org/fx"

var Module = fx.Module("imaging", fx.Provide(New))
