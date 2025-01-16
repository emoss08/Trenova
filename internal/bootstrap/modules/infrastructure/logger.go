package infrastructure

import (
	"github.com/trenova-app/transport/internal/pkg/logger"
	"go.uber.org/fx"
)

var LoggerModule = fx.Module("logger", fx.Provide(logger.NewLogger))
