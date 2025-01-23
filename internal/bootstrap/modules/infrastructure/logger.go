package infrastructure

import (
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.uber.org/fx"
)

var LoggerModule = fx.Module("logger", fx.Provide(logger.NewLogger))
