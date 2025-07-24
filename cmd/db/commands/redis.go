/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package commands

import (
	"context"
	"log"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Redis CLI",
	Long:  `A complete CLI for managing the Redis database.`,
}

func init() {
	rootCmd.AddCommand(redisCmd)
	redisCmd.AddCommand(FlushAllCmd)
}

var FlushAllCmd = &cobra.Command{
	Use:   "flushall",
	Short: "Flush all data from Redis",
	Run: func(cmd *cobra.Command, args []string) {
		GetRedisClient().FlushAll(context.Background())
	},
}

func GetRedisClient() *redis.Client {
	manager := config.NewManager()

	cfg, err := manager.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	opts := &redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	redisClient := redis.NewClient(opts)

	return redisClient
}
