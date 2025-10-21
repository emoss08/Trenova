package providers

import (
	"go.uber.org/fx"
)

var Module = fx.Module("email_providers",
	fx.Provide(
		fx.Annotate(
			NewSMTPProvider,
			fx.As(new(Provider)),
			fx.ResultTags(`group:"email_providers"`),
		),
		fx.Annotate(
			NewResendProvider,
			fx.As(new(Provider)),
			fx.ResultTags(`group:"email_providers"`),
		),
	),

	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(Registry)),
		),
	),
)
