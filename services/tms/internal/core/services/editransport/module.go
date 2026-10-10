package editransport

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"go.uber.org/fx"
)

var Module = fx.Module("edi-transport",
	fx.Provide(
		fx.Annotate(
			NewSFTPTransport,
			fx.As(new(ediports.EDITransport)),
			fx.ResultTags(`group:"edi_transports"`),
		),
		fx.Annotate(
			NewVANTransport,
			fx.As(new(ediports.EDITransport)),
			fx.ResultTags(`group:"edi_transports"`),
		),
		fx.Annotate(
			NewAS2Transport,
			fx.As(new(ediports.EDITransport)),
			fx.ResultTags(`group:"edi_transports"`),
		),
		fx.Annotate(
			NewDispatcher,
			fx.As(new(ediports.EDITransportDispatcher)),
		),
	),
)
