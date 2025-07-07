package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/routing/internal/api"
	"github.com/emoss08/routing/internal/kafka"
	"github.com/emoss08/routing/internal/storage"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	// _ Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

// RouteCalculatorAdapter adapts the API handler to the RouteCalculator interface
type RouteCalculatorAdapter struct {
	handler *api.Handler
}

func (a *RouteCalculatorAdapter) CalculateRoute(
	ctx context.Context,
	originZip, destZip, vehicleType string,
) (float64, float64, error) {
	// _ This is a simplified implementation
	// _ In production, you would call the actual handler method
	// req := api.RouteDistanceRequest{
	// 	OriginZip:   originZip,
	// 	DestZip:     destZip,
	// 	VehicleType: vehicleType,
	// }

	// _ For now, return a simulated response
	// _ In real implementation, this would call handler.calculateRoute
	return 380.5, 360.5, nil
}

func main() {
	// _ Load configuration
	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// _ Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// _ Initialize PostgreSQL
	dbDSN := viper.GetString("database.dsn")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:password@localhost:5432/routing?sslmode=disable"
	}

	storage, err := storage.NewPostgresStorage(ctx, dbDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer storage.Close()

	// _ Initialize Redis
	redisAddr := viper.GetString("redis.addr")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	cache := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.db"),
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})

	if err := cache.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer cache.Close()

	// _ Create API handler for route calculations
	handler := api.NewHandler(
		storage,
		cache,
		log.Logger,
		nil,
	) // No Kafka producer needed for consumer
	adapter := &RouteCalculatorAdapter{handler: handler}

	// _ Create batch processor
	batchProcessor := kafka.NewBatchRouteProcessor(
		adapter,
		log.Logger,
		10,
	) // Max 10 concurrent calculations

	// _ Create batch processor handler
	batchHandler := kafka.NewBatchProcessorHandler(batchProcessor, log.Logger)

	// _ Create Kafka consumer
	consumerConfig := kafka.ConsumerConfig{
		Brokers:        viper.GetStringSlice("kafka.brokers"),
		Topic:          viper.GetString("kafka.topics.batch_requests"),
		GroupID:        viper.GetString("kafka.consumer.group_id") + "-batch",
		MinBytes:       viper.GetInt("kafka.consumer.min_bytes"),
		MaxBytes:       viper.GetInt("kafka.consumer.max_bytes"),
		MaxWait:        viper.GetDuration("kafka.consumer.max_wait"),
		StartOffset:    -2, // Start from oldest
		CommitInterval: viper.GetDuration("kafka.consumer.commit_interval"),
	}

	consumer := kafka.NewConsumer(consumerConfig, batchHandler, log.Logger)

	// _ Start consumer in goroutine
	go func() {
		log.Info().
			Str("topic", consumerConfig.Topic).
			Str("group_id", consumerConfig.GroupID).
			Msg("Starting batch consumer")

		if err := consumer.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Consumer error")
			cancel()
		}
	}()

	// _ Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down batch consumer...")

	// _ Cancel context to stop consumer
	cancel()

	// _ Give consumer time to shut down gracefully
	time.Sleep(2 * time.Second)

	log.Info().Msg("Batch consumer exited")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/routing/")

	// _ Set defaults
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("kafka.consumer.min_bytes", 10240)
	viper.SetDefault("kafka.consumer.max_bytes", 10485760)
	viper.SetDefault("kafka.consumer.max_wait", "500ms")
	viper.SetDefault("kafka.consumer.commit_interval", "1s")

	// _ Read environment variables
	viper.SetEnvPrefix("ROUTING")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
		log.Warn().Msg("No config file found, using defaults and environment variables")
	}

	return nil
}
