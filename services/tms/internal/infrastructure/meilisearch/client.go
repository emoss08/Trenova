package meilisearch

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	meili "github.com/meilisearch/meilisearch-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
	LC     fx.Lifecycle
}

type Client struct {
	client  meili.ServiceManager
	enabled bool
	logger  *zap.Logger
}

var _ repositories.SearchRepository = (*Client)(nil)

func New(p Params) (*Client, error) {
	searchConfig := p.Config.GetSearchConfig()
	logger := p.Logger.Named("meilisearch")

	client := &Client{
		enabled: searchConfig.Enabled,
		logger:  logger,
	}

	if !searchConfig.Enabled {
		return client, nil
	}

	meiliClient := meili.New(
		searchConfig.Meilisearch.URL,
		meili.WithAPIKey(searchConfig.Meilisearch.APIKey),
		meili.WithCustomClient(&http.Client{
			Timeout: searchConfig.Meilisearch.GetTimeout(),
		}),
	)
	client.client = meiliClient

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			_, err := meiliClient.Health()
			if err != nil {
				return fmt.Errorf("ping meilisearch: %w", err)
			}
			logger.Info(
				"Meilisearch connection established",
				zap.String("url", searchConfig.Meilisearch.URL),
			)
			return nil
		},
		OnStop: func(context.Context) error {
			meiliClient.Close()
			return nil
		},
	})

	return client, nil
}

func (c *Client) Enabled() bool {
	return c.enabled
}

func (c *Client) Search(
	ctx context.Context,
	req repositories.SearchRequest,
) ([]map[string]any, error) {
	if !c.Enabled() {
		return nil, repositories.ErrCacheMiss
	}

	result, err := c.client.Index(req.Index).Search(req.Query, &meili.SearchRequest{
		Limit:  int64(req.Limit),
		Filter: req.Filter,
	})
	if err != nil {
		return nil, fmt.Errorf("search index %s: %w", req.Index, err)
	}

	documents := make([]map[string]any, 0, len(result.Hits))
	for _, hit := range result.Hits {
		document := make(map[string]any, len(hit))
		for key, value := range hit {
			document[key] = value
		}
		documents = append(documents, document)
	}

	return documents, nil
}
