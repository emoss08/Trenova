package aiproviderservice

import "go.uber.org/fx"

var Module = fx.Module("aiprovider-service",
	fx.Provide(
		NewProber,
		New,
	),
)
