/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/routing/internal/graph"
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

	// _ Initialize Redis for cache invalidation
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

	// _ Load graph for updates
	log.Info().Msg("Loading routing graph...")
	g, err := storage.LoadGraphForRegion(ctx, 32.0, -125.0, 42.0, -114.0) // California region
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load graph")
	}

	router := graph.NewRouter(g)
	updateService := kafka.NewGraphUpdateService(router, log.Logger)

	// _ Create data update handler
	dataHandler := kafka.NewDataUpdateHandler(updateService, log.Logger)

	// _ Create consumers for different update topics
	consumers := []struct {
		topic   string
		handler kafka.MessageHandler
	}{
		{
			topic:   viper.GetString("kafka.topics.osm_updates"),
			handler: dataHandler,
		},
		{
			topic:   viper.GetString("kafka.topics.restriction_updates"),
			handler: dataHandler,
		},
	}

	// _ Start consumers
	for _, config := range consumers {
		consumerConfig := kafka.ConsumerConfig{
			Brokers:        viper.GetStringSlice("kafka.brokers"),
			Topic:          config.topic,
			GroupID:        viper.GetString("kafka.consumer.group_id") + "-data",
			MinBytes:       viper.GetInt("kafka.consumer.min_bytes"),
			MaxBytes:       viper.GetInt("kafka.consumer.max_bytes"),
			MaxWait:        viper.GetDuration("kafka.consumer.max_wait"),
			StartOffset:    -2, // Start from oldest
			CommitInterval: viper.GetDuration("kafka.consumer.commit_interval"),
		}

		consumer := kafka.NewConsumer(consumerConfig, config.handler, log.Logger)

		go func(c *kafka.Consumer, topic string) {
			log.Info().
				Str("topic", topic).
				Str("group_id", consumerConfig.GroupID).
				Msg("Starting data update consumer")

			if err := c.Start(ctx); err != nil {
				log.Error().Err(err).Str("topic", topic).Msg("Consumer error")
				cancel()
			}
		}(consumer, config.topic)
	}

	// _ Create cache invalidation producer
	cacheProducerConfig := kafka.ProducerConfig{
		Brokers:      viper.GetStringSlice("kafka.brokers"),
		Topic:        viper.GetString("kafka.topics.cache_invalidation"),
		BatchSize:    viper.GetInt("kafka.producer.batch_size"),
		BatchTimeout: viper.GetDuration("kafka.producer.batch_timeout"),
		Async:        true,
		Compression:  viper.GetString("kafka.producer.compression"),
	}

	cacheProducer := kafka.NewProducer(cacheProducerConfig, log.Logger)
	defer cacheProducer.Close()

	// _ Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down data consumer...")

	// _ Cancel context to stop consumers
	cancel()

	// _ Give consumers time to shut down gracefully
	time.Sleep(2 * time.Second)

	log.Info().Msg("Data consumer exited")
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
