package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/database/dbbackup"
	"go.uber.org/fx"
)

var Module = fx.Module("infrastructure",
	ConfigModule,
	LoggerModule,
	DatabaseModule,
	StorageModule,
	CacheModule,
	SearchModule,
	MessagingModule,
)

var BackupModule = fx.Module("db_backup",
	fx.Provide(dbbackup.NewBackupService),
	fx.Provide(dbbackup.NewBackupScheduler),
)
