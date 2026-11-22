package redis

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/cacheconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/startwait"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/redis"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPoolSize     = 25
	defaultMinIdleConns = 10
	defaultDialTimeout  = 10 * time.Second
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 5 * time.Second
	defaultMaxRetries   = 3
)

type ConnectionParams struct {
	fx.In

	CacheConfig      *cacheconfig.Config
	MonitoringConfig *monitoringconfig.Config
	Logger           *zap.Logger
	LC               fx.Lifecycle
}

func NewConnection(p ConnectionParams) (*redis.Client, error) {
	logger := p.Logger.With(zap.String("component", "redis"))
	cacheConfig := p.CacheConfig

	poolSize := intutils.WithDefault(cacheConfig.PoolSize, defaultPoolSize)

	options := &redis.Options{
		Addr:            cacheConfig.GetRedisAddr(),
		Username:        cacheConfig.Username,
		Password:        cacheConfig.Password,
		DB:              cacheConfig.DB,
		PoolSize:        poolSize,
		MinIdleConns:    intutils.WithDefault(cacheConfig.MinIdleConns, defaultMinIdleConns),
		ConnMaxIdleTime: cacheConfig.ConnMaxIdleTime,
		ConnMaxLifetime: cacheConfig.ConnMaxLifetime,
		PoolTimeout:     cacheConfig.PoolTimeout,
		DialTimeout:     timeutils.WithDefaultDuration(cacheConfig.DialTimeout, defaultDialTimeout),
		ReadTimeout:     timeutils.WithDefaultDuration(cacheConfig.ReadTimeout, defaultReadTimeout),
		WriteTimeout: timeutils.WithDefaultDuration(
			cacheConfig.WriteTimeout,
			defaultWriteTimeout,
		),
		MaxRetries:      intutils.WithDefault(cacheConfig.MaxRetries, defaultMaxRetries),
		MinRetryBackoff: cacheConfig.MinRetryBackoff,
		MaxRetryBackoff: cacheConfig.MaxRetryBackoff,
	}
	if cacheConfig.TLS.Enabled {
		tlsConfig, err := cacheConfig.TLSClientConfig()
		if err != nil {
			return nil, err
		}
		options.TLSConfig = tlsConfig
	}

	if p.MonitoringConfig.Tracing.Enabled {
		options.TracerProvider = otel.GetTracerProvider()
		options.MeterProvider = otel.GetMeterProvider()
	}

	client := redis.NewClient(options)

	p.LC.Append(fx.Hook{
		OnStop: func(context.Context) error {
			logger.Info("Closing Redis connection")
			if err := client.Close(); err != nil {
				logger.Error("Failed to close Redis connection", zap.Error(err))
				return err
			}

			logger.Info("Redis connection closed successfully")
			return nil
		},
	})

	return client, nil
}

type AwaitParams struct {
	fx.In

	Client      *redis.Client
	CacheConfig *cacheconfig.Config
	AppConfig   *appconfig.Config
	Logger      *zap.Logger
}

func AwaitReady(p AwaitParams) error {
	logger := p.Logger.With(zap.String("component", "redis"))

	err := startwait.Wait(context.Background(), startwait.Params{
		Logger:     logger,
		Dependency: "Redis",
		Deadline:   startwait.Deadline(p.AppConfig.StartupWait),
		Check: func(ctx context.Context) error {
			return p.Client.Ping(ctx).Err()
		},
	})
	if err != nil {
		return err
	}

	logger.Info(
		"Redis connection established",
		zap.String("host", p.CacheConfig.Host),
		zap.Int("port", p.CacheConfig.Port),
		zap.Int("db", p.CacheConfig.DB),
		zap.Int("poolSize", p.Client.Options().PoolSize),
	)
	return nil
}
