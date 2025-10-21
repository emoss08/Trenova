package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap/modules/infrastructure"
	infraConfig "github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RedisParams struct {
	fx.In
	RedisConnection *redis.Connection
	Logger          *zap.Logger
	Shutdowner      fx.Shutdowner
}

var redisFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush the Redis cache",
	RunE:  runRedisFlush,
}

func flushRedis(params RedisParams) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Flush Redis
	err := params.RedisConnection.Client().FlushAll(ctx).Err()
	if err != nil {
		params.Logger.Error("Failed to flush Redis", zap.Error(err))
		return err
	}

	params.Logger.Info("Successfully flushed Redis cache")
	fmt.Println("✓ Redis cache flushed successfully")

	// Shutdown the app after completion
	return params.Shutdowner.Shutdown()
}

func runRedisFlush(cmd *cobra.Command, args []string) error {
	// Create a minimal fx app with only required dependencies
	app := fx.New(
		// Provide config
		fx.Provide(func() *infraConfig.Config {
			return cfg
		}),
		// Provide logger
		fx.Provide(infraConfig.ProvideLogger),
		// Only include Redis module
		infrastructure.RedisModule,
		// Invoke the flush function
		fx.Invoke(flushRedis),
		// Start and stop immediately
		fx.StopTimeout(15*time.Second),
	)

	// Run the app
	if err := app.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// The app will shutdown automatically after flushRedis completes
	<-app.Done()

	return app.Err()
}

func init() {
	RedisCmd.AddCommand(redisFlushCmd)
}
