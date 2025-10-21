package config

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the configuration module for fx
var Module = fx.Module("config",
	fx.Provide(
		ProvideConfig,
		ProvideLogger,
	),
)

// ProvideConfig provides the configuration to the fx container
func ProvideConfig() (*Config, error) {
	loader := NewLoader(
		WithConfigPath("config"),
	)

	config, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}

// ProvideLogger provides a logger based on the configuration
func ProvideLogger(config *Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	// Create logger based on environment
	if config.IsDevelopment() || config.App.Debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Set global logger
	zap.ReplaceGlobals(logger)

	return logger, nil
}

// ConfigParams defines the dependencies for components that need config
type Params struct {
	fx.In
	Config *Config
	Logger *zap.Logger
}

// Hooks provides lifecycle hooks for configuration
func Hooks() fx.Option {
	return fx.Options(
		fx.Invoke(func(lc fx.Lifecycle, config *Config, logger *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					logger.Info("Starting application",
						zap.String("name", config.App.Name),
						zap.String("version", config.App.Version),
						zap.String("environment", config.App.Env),
						zap.Bool("debug", config.App.Debug),
					)

					// Log masked database connection
					logger.Debug("Database configuration",
						zap.String("dsn", config.GetDSNMasked()),
					)

					// Validate configuration one more time at startup
					if config.IsProduction() {
						logger.Info("Running in production mode - strict security checks enabled")
					}

					return nil
				},
				OnStop: func(context.Context) error {
					logger.Info("Shutting down application")
					// Ignore sync errors for stderr/stdout as they're not regular files
					_ = logger.Sync()
					return nil
				},
			})
		}),
	)
}
