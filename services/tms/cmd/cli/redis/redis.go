package redis

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

func SetConfig(c *config.Config) {
	cfg = c
}

var RedisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Redis management commands",
}
