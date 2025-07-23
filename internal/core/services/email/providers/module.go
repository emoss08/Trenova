// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package providers

import (
	"go.uber.org/fx"
)

// Module provides all email providers
var Module = fx.Module("email_providers",
	// Provide individual providers
	fx.Provide(
		fx.Annotate(
			NewSMTPProvider,
			fx.As(new(Provider)),
			fx.ResultTags(`group:"email_providers"`),
		),
		fx.Annotate(
			NewSendGridProvider,
			fx.As(new(Provider)),
			fx.ResultTags(`group:"email_providers"`),
		),
		// Add more providers here as they are implemented
		// fx.Annotate(
		//     NewAWSSESProvider,
		//     fx.As(new(Provider)),
		//     fx.ResultTags(`group:"email_providers"`),
		// ),
	),

	// Provide the registry
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(Registry)),
		),
	),
)
