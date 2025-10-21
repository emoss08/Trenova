package redis

import (
	infraConfig "github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var (
	cfg *infraConfig.Config
)

// SetConfig sets the config for the redis package
func SetConfig(c *infraConfig.Config) {
	cfg = c
}

var RedisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Redis management commands",
	Long:  `Redis management commands for cache operations.`,
}