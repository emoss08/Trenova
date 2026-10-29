package editransport

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"go.uber.org/fx"
)

var Module = fx.Module("edi-transport",
	fx.Provide(
		fx.Annotate(
			NewSFTPTransport,
			fx.As(new(editransportports.EDITransport)),
			fx.ResultTags(`group:"edi_transports"`),
		),
		fx.Annotate(
			NewVANTransport,
			fx.As(new(editransportports.EDITransport)),
			fx.ResultTags(`group:"edi_transports"`),
		),
		fx.Annotate(
			NewAS2Transport,
			fx.As(new(editransportports.EDITransport)),
			fx.ResultTags(`group:"edi_transports"`),
		),
		fx.Annotate(
			NewDispatcher,
			fx.As(new(editransportports.EDITransportDispatcher)),
		),
	),
)
