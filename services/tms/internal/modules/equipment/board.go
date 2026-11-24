package equipment

import (
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentboardreader"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"go.uber.org/fx"
)

var boardReaders = fx.Options(
	fx.Provide(
		fx.Annotate(
			equipmentboardreader.NewTractors,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			equipmentboardreader.NewTrailers,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
)
