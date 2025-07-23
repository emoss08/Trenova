// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/storage/minio"
	"go.uber.org/fx"
)

var StorageModule = fx.Module("storage", fx.Provide(minio.NewClient))
