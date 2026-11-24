package parties

import (
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/partiesboardreader"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"go.uber.org/fx"
)

var boardReaders = fx.Options(
	fx.Provide(
		fx.Annotate(
			partiesboardreader.NewCarriers,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			partiesboardreader.NewCustomers,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
)
