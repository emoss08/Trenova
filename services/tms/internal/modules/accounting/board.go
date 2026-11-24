package accounting

import (
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/accountingboardreader"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"go.uber.org/fx"
)

var boardReaders = fx.Options(
	fx.Provide(
		fx.Annotate(
			accountingboardreader.NewEntries,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			accountingboardreader.NewAmount,
			fx.As(new(boardreaderports.MeasureReader)),
			fx.ResultTags(boardreaderports.MeasureReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			accountingboardreader.NewPendingFor,
			fx.As(new(boardreaderports.MeasureReader)),
			fx.ResultTags(boardreaderports.MeasureReaders),
		),
	),
	fx.Provide(
		func(s *fiscalperiodservice.Service) fiscalperiodports.CloseBlockerReader { return s },
	),
	fx.Provide(
		fx.Annotate(
			accountingboardreader.NewPeriodClose,
			fx.As(new(boardreaderports.MetricReader)),
			fx.ResultTags(boardreaderports.MetricReaders),
		),
	),
)
