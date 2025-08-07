/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package infrastructure

import (
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.uber.org/fx"
)

var LoggerModule = fx.Module("logger", fx.Provide(logger.NewLogger))
