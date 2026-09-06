package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Module = fx.Module("config",
	fx.Provide(
		ProvideConfig,
		ProvideLogger,
	),
)

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

// LogSink owns whatever the logger writes to, so the application can release
// it on the way out. Logging to stdout or stderr has nothing to close; logging
// to a file holds a handle open for the life of the process, and closing it is
// what flushes the last of the buffer to disk.
type LogSink struct {
	closer io.Closer
}

func (s *LogSink) Close() error {
	if s == nil || s.closer == nil {
		return nil
	}
	return s.closer.Close()
}

func ProvideLogger(config *Config) (*zap.Logger, *LogSink, error) {
	logger, sink, err := newLogger(config.Logging)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}

	zap.ReplaceGlobals(logger)

	return logger, sink, nil
}

func newLogger(cfg LoggingConfig) (*zap.Logger, *LogSink, error) {
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	encoder, err := newLogEncoder(cfg.Format)
	if err != nil {
		return nil, nil, err
	}

	sink, closer, err := newLogSink(cfg)
	if err != nil {
		return nil, nil, err
	}

	core := zapcore.NewCore(encoder, sink, level)
	if cfg.Sampling {
		core = zapcore.NewSamplerWithOptions(core, time.Second, 100, 100)
	}

	options := []zap.Option{
		zap.AddCaller(),
		zap.ErrorOutput(sink),
	}
	if cfg.Stacktrace {
		options = append(options, zap.AddStacktrace(zapcore.WarnLevel))
	}

	return zap.New(core, options...), &LogSink{closer: closer}, nil
}

func parseLogLevel(level string) (zapcore.Level, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return 0, fmt.Errorf("invalid log level %q: %w", level, err)
	}
	return zapLevel, nil
}

func newLogEncoder(format string) (zapcore.Encoder, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	switch format {
	case "json":
		return zapcore.NewJSONEncoder(encoderConfig), nil
	case "text":
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	default:
		return nil, fmt.Errorf("invalid log format %q", format)
	}
}

// newLogSink returns where the logger writes and, when that is something the
// process holds open, the closer for it. Stdout and stderr belong to the
// process rather than to us, so they come back with no closer.
func newLogSink(cfg LoggingConfig) (zapcore.WriteSyncer, io.Closer, error) {
	switch cfg.Output {
	case "stdout":
		return zapcore.Lock(os.Stdout), nil, nil
	case "stderr":
		return zapcore.Lock(os.Stderr), nil, nil
	case "file":
		if cfg.File == nil {
			return nil, nil, ErrLoggingOutputIsFileButFileConfigIsMissing
		}

		dir := filepath.Dir(cfg.File.Path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return nil, nil, fmt.Errorf("create log directory: %w", err)
			}
		}

		rotator := &lumberjack.Logger{
			Filename:   cfg.File.Path,
			MaxSize:    cfg.File.MaxSize,
			MaxAge:     cfg.File.MaxAge,
			MaxBackups: cfg.File.MaxBackups,
			Compress:   cfg.File.Compress,
		}
		return zapcore.AddSync(rotator), rotator, nil
	default:
		return nil, nil, fmt.Errorf("invalid log output %q", cfg.Output)
	}
}

type Params struct {
	fx.In
	Config *Config
	Logger *zap.Logger
}

func Hooks() fx.Option {
	return fx.Options(
		fx.Invoke(func(lc fx.Lifecycle, config *Config, logger *zap.Logger, sink *LogSink) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					logger.Debug("Database configuration",
						zap.String("dsn", config.GetDSNMasked()),
					)

					logger.Info("Starting application",
						zap.String("name", config.App.Name),
						zap.String("version", config.App.Version),
						zap.String("environment", config.App.Env),
						zap.Bool("debug", config.App.Debug),
					)

					if config.App.IsProduction() {
						logger.Info("Running in production mode - strict security checks enabled")
					}

					return nil
				},
				OnStop: func(context.Context) error {
					logger.Info("Shutting down application")
					// Ignore sync errors for stderr/stdout as they're not regular files
					_ = logger.Sync()
					// Closing the sink is what releases a log file. Syncing
					// alone leaves the handle open for the life of the process.
					return sink.Close()
				},
			})
		}),
	)
}
