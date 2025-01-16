package infrastructure

import (
	"github.com/trenova-app/transport/internal/infrastructure/storage/minio"
	"go.uber.org/fx"
)

var StorageModule = fx.Module("storage", fx.Provide(minio.NewClient))
