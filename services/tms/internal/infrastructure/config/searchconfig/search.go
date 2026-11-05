package searchconfig

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
)

var ErrProductionAPIKeyRequired = errors.New(
	"production and staging require search.meilisearch.apiKey that is not a placeholder",
)

type Config struct {
	Enabled      bool              `mapstructure:"enabled"`
	DefaultLimit int               `mapstructure:"defaultLimit" validate:"omitempty,min=1,max=100"`
	Meilisearch  MeilisearchConfig `mapstructure:"meilisearch"`
}

func (c *Config) ValidateProduction() error {
	if c.Enabled && securityconfig.IsPlaceholder(c.Meilisearch.APIKey) {
		return ErrProductionAPIKeyRequired
	}

	return nil
}

func (c *Config) GetDefaultLimit() int {
	if c.DefaultLimit <= 0 {
		return 8
	}
	return c.DefaultLimit
}

type MeilisearchConfig struct {
	URL     string                 `mapstructure:"url"     validate:"omitempty,url,no_trailing_slash"`
	APIKey  string                 `mapstructure:"apiKey"`
	Timeout time.Duration          `mapstructure:"timeout"`
	Indexes MeilisearchIndexConfig `mapstructure:"indexes"`
}

func (c *MeilisearchConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 3 * time.Second
	}

	return c.Timeout
}

type MeilisearchIndexConfig struct {
	Shipments string `mapstructure:"shipments"`
	Customers string `mapstructure:"customers"`
	Workers   string `mapstructure:"workers"`
	Documents string `mapstructure:"documents"`
	Invoices  string `mapstructure:"invoices"`
}

func Defaults() map[string]any {
	return map[string]any{
		"enabled":                       false,
		"defaultLimit":                  8,
		"meilisearch.url":               "http://localhost:7700",
		"meilisearch.timeout":           "3s",
		"meilisearch.indexes.shipments": "shipments",
		"meilisearch.indexes.customers": "customers",
		"meilisearch.indexes.workers":   "workers",
		"meilisearch.indexes.documents": "documents",
		"meilisearch.indexes.invoices":  "invoices",
	}
}
