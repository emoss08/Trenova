package partnerconfig

import "go.uber.org/fx"

var Module = fx.Module(
	"edi-partner-config",
	fx.Provide(
		NewServer,
		NewPartnerConfigClient,
	),
)
