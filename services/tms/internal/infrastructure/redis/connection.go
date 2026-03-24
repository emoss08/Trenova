package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
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

	Config *config.Config
	Logger *zap.Logger
	Tracer observability.Tracer `optional:"true"`
	LC     fx.Lifecycle
}

func NewConnection(p ConnectionParams) (*redis.Client, error) {
	logger := p.Logger.With(zap.String("component", "redis"))
	cacheConfig := p.Config.GetCacheConfig()

	poolSize := intutils.WithDefault(cacheConfig.PoolSize, defaultPoolSize)

	client := redis.NewClient(&redis.Options{
		Addr:            cacheConfig.GetRedisAddr(),
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
	})

	if p.Tracer != nil && p.Tracer.IsEnabled() {
		if err := redisotel.InstrumentTracing(client); err != nil {
			logger.Warn("Failed to instrument Redis tracing", zap.Error(err))
		}
		if err := redisotel.InstrumentMetrics(client); err != nil {
			logger.Warn("Failed to instrument Redis metrics", zap.Error(err))
		}
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("failed to ping Redis: %w", err)
			}
			logger.Info("Redis connection established",
				zap.String("host", cacheConfig.Host),
				zap.Int("port", cacheConfig.Port),
				zap.Int("db", cacheConfig.DB),
				zap.Int("poolSize", poolSize),
			)
			return nil
		},
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
