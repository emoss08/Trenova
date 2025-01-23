package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/storage/minio"
	"go.uber.org/fx"
)

var StorageModule = fx.Module("storage", fx.Provide(minio.NewClient))
