package config

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func FxLogger() fx.Option {
	return fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
		fxLogger := &fxevent.ZapLogger{Logger: logger.Named("fx")}
		fxLogger.UseLogLevel(zapcore.DebugLevel)
		fxLogger.UseErrorLevel(zapcore.ErrorLevel)

		return fxLogger
	})
}
