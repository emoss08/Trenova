package workspace

import (
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/homeboardreader"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"go.uber.org/fx"
)

var boardReaders = fx.Options(
	fx.Provide(
		fx.Annotate(
			homeboardreader.NewDispatch,
			fx.As(new(boardreaderports.MetricReader)),
			fx.ResultTags(boardreaderports.MetricReaders),
		),
	),
)
