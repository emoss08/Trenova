package meilisearch

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultTimeout     = 10 * time.Second
	defaultIndexPrefix = "trenova"
)

type ConnectionParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Connection struct {
	mgr         meilisearch.ServiceManager
	cfg         *config.SearchConfig
	logger      *zap.Logger
	indexPrefix string
}

func NewConnection(p ConnectionParams) (*Connection, error) {
	logger := p.Logger.With(zap.String("component", "meilisearch"))
	cfg := p.Config.Search

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	indexPrefix := cfg.IndexPrefix
	if indexPrefix == "" {
		indexPrefix = defaultIndexPrefix
	}

	mgr := meilisearch.New(cfg.Host, meilisearch.WithAPIKey(cfg.APIKey))

	conn := &Connection{
		mgr:         mgr,
		cfg:         cfg,
		logger:      logger,
		indexPrefix: indexPrefix,
	}

	logger.Info("Meilisearch connection established",
		zap.String("host", cfg.Host),
		zap.String("indexPrefix", indexPrefix),
		zap.Duration("timeout", timeout),
	)

	return conn, nil
}

func (c *Connection) Manager() meilisearch.ServiceManager {
	return c.mgr
}
