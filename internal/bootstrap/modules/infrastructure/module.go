package infrastructure

import (
	"go.uber.org/fx"
)

var Module = fx.Module("infrastructure",
	ConfigModule,
	LoggerModule,
	DatabaseModule,
	StorageModule,
	CacheModule,
	SearchModule,
	RoutingModule,
)
