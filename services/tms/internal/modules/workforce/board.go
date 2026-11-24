package workforce

import (
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workforceboardreader"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"go.uber.org/fx"
)

var boardReaders = fx.Options(
	fx.Provide(
		fx.Annotate(
			workforceboardreader.NewWorkers,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			workforceboardreader.NewCredentials,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			workforceboardreader.NewExpiry,
			fx.As(new(boardreaderports.MeasureReader)),
			fx.ResultTags(boardreaderports.MeasureReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			workforceboardreader.NewHoursThisWeek,
			fx.As(new(boardreaderports.MeasureReader)),
			fx.ResultTags(boardreaderports.MeasureReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			workforceboardreader.NewRotaWeek,
			fx.As(new(boardreaderports.MetricReader)),
			fx.ResultTags(boardreaderports.MetricReaders),
		),
	),
)
