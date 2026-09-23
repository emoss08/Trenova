package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/minio"
	"go.temporal.io/sdk/converter"
	"go.uber.org/fx"
)

var StorageModule = fx.Module("storage", fx.Provide(
	minio.New,
	fx.Annotate(
		minio.NewTemporalPayloadStore,
		fx.As(new(converter.StorageDriver)),
	),
))
