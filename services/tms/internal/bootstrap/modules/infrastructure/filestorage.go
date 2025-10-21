package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/filestorage"
	"go.uber.org/fx"
)

var FileStorageModule = fx.Module("filestorage",
	fx.Provide(filestorage.NewClient),
)
