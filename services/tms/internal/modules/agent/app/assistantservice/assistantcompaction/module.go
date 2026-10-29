package assistantcompaction

import (
	"go.uber.org/fx"
)

var Module = fx.Module("assistant-compaction", fx.Provide(New))
