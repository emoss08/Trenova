package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	infraConfig "github.com/emoss08/trenova/internal/infrastructure/config"
	goredis "github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RedisFlushParams struct {
	fx.In
	Client     *goredis.Client
	Logger     *zap.Logger
	Shutdowner fx.Shutdowner
}

var redisFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush the Redis cache",
	RunE:  runRedisFlush,
}

func flushRedis(params RedisFlushParams) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := params.Client.FlushAll(ctx).Err(); err != nil {
		params.Logger.Error("Failed to flush Redis", zap.Error(err))
		return err
	}

	params.Logger.Info("Successfully flushed Redis cache")
	fmt.Println("Redis cache flushed successfully")

	return params.Shutdowner.Shutdown()
}

func runRedisFlush(cmd *cobra.Command, args []string) error {
	app := fx.New(
		fx.Provide(func() *infraConfig.Config {
			return cfg
		}),
		fx.Provide(infraConfig.ProvideLogger),
		infrastructure.RedisModule,
		fx.Invoke(flushRedis),
		fx.StopTimeout(15*time.Second),
	)

	if err := app.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	<-app.Done()

	return app.Err()
}

func init() {
	RedisCmd.AddCommand(redisFlushCmd)
}
