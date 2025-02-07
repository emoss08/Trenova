package meilisearch

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/meilisearch/meilisearch-go"
	"github.com/mitchellh/mapstructure"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ClientParams struct {
	fx.In

	Config *config.Manager
	Logger *logger.Logger
}

type client struct {
	meili meilisearch.ServiceManager
	l     *zerolog.Logger
	cfg   *config.SearchConfig
}

func NewClient(p ClientParams) infra.SearchClient {
	log := p.Logger.With().Str("client", "meilisearch").Logger()

	cfg := p.Config.Search()
	if cfg == nil {
		log.Error().Msg("config is nil")
		return nil
	}

	log.Debug().
		Str("host", cfg.Host).
		Int("maxBatchSize", cfg.MaxBatchSize).
		Int("batchInterval", cfg.BatchInterval).
		Str("indexPrefix", cfg.IndexPrefix).
		Msg("initializing search client")

	c := &client{
		meili: meilisearch.New(cfg.Host, meilisearch.WithAPIKey(cfg.APIKey)),
		l:     &log,
		cfg:   cfg,
	}

	// TODO(wolfred): initialize the indexes
	if err := c.InitializeIndexes(); err != nil {
		log.Error().Err(err).Msg("failed to initialize indexes")
	}

	return c
}

func (c *client) InitializeIndexes() error {
	idxName := c.GetIndexName()
	c.l.Debug().Str("idxName", idxName).Msg("initializing index")

	// configure the global settings for the index
	settings := &meilisearch.Settings{
		SearchableAttributes: []string{"title", "description", "searchableText"},
		FilterableAttributes: []string{"type", "organizationId", "businessUnitId", "createdAt", "updatedAt"},
		SortableAttributes:   []string{"createdAt", "updatedAt"},
		RankingRules:         []string{"words", "typo", "proximity", "sort", "attribute", "exactness"},
	}

	// create or update the index
	_, err := c.meili.GetIndex(idxName)
	if err != nil {
		c.l.Debug().Err(err).Str("idxName", idxName).Msg("index not found, creating...")
		_, err = c.meili.CreateIndex(&meilisearch.IndexConfig{
			Uid:        idxName,
			PrimaryKey: "id",
		})
		if err != nil {
			c.l.Error().Err(err).Str("idxName", idxName).Msg("failed to create index")
			return eris.Wrap(err, "failed to create index")
		}
	}

	// update the index settings
	_, err = c.meili.Index(idxName).UpdateSettings(settings)
	if err != nil {
		c.l.Error().Err(err).Str("idxName", idxName).Msg("failed to update index settings")
		return eris.Wrap(err, "failed to update index settings")
	}

	c.l.Debug().Str("idxName", idxName).Msg("index initialized")
	return nil
}

func (c *client) IndexDocuments(indexName string, docs []*infra.SearchDocument) (*infra.SearchTaskInfo, error) {
	tInfo, err := c.meili.Index(indexName).AddDocuments(docs)
	if err != nil {
		return nil, eris.Wrap(err, "failed to index documents")
	}

	return &infra.SearchTaskInfo{
		UID:        tInfo.TaskUID,
		IndexUID:   tInfo.IndexUID,
		Type:       string(tInfo.Type),
		EnqueuedAt: tInfo.EnqueuedAt.Format(time.RFC3339),
	}, nil
}

func (c *client) WaitForTask(taskUID int64, timeout time.Duration) (*infra.SearchTask, error) {
	task, err := c.meili.WaitForTask(taskUID, timeout)
	if err != nil {
		return nil, eris.Wrap(err, "wait for task")
	}

	return &infra.SearchTask{
		Status:     string(task.Status),
		TaskUID:    task.TaskUID,
		IndexUID:   task.IndexUID,
		Type:       string(task.Type),
		EnqueuedAt: task.EnqueuedAt,
		Duration:   task.Duration,
		StartedAt:  task.StartedAt,
		FinishedAt: task.FinishedAt,
		CanceledBy: task.CanceledBy,
		Error: infra.SearchTaskError{
			Message: task.Error.Message,
			Code:    task.Error.Code,
			Type:    task.Error.Type,
			Link:    task.Error.Link,
		},
	}, nil
}

func (c *client) Search(ctx context.Context, opts *infra.SearchOptions) ([]*infra.SearchDocument, error) {
	searchRequest := &meilisearch.SearchRequest{
		Query:  opts.Query,
		Limit:  int64(opts.Limit),
		Offset: int64(opts.Offset),
		Filter: fmt.Sprintf("businessUnitId = %s AND organizationId = %s", opts.BuID, opts.OrgID),
		Sort:   opts.SortBy,
	}

	// perform the search
	c.l.Debug().Interface("searchRequest", searchRequest).Msg("executing search request")
	res, err := c.meili.Index(c.GetIndexName()).SearchWithContext(ctx, opts.Query, searchRequest)
	if err != nil {
		c.l.Error().Err(err).Msg("failed to execute search request")
		return nil, eris.Wrap(err, "failed to execute search request")
	}

	// parse the results
	results := make([]*infra.SearchDocument, 0)

	// iterate over the hits and convert them to documents
	for _, hit := range res.Hits {
		if doc, ok := hit.(map[string]any); ok {
			var searchDoc infra.SearchDocument

			// convert map to documnet
			if err = mapstructure.Decode(doc, &searchDoc); err != nil {
				c.l.Error().Err(err).Msg("failed to decode search result")
				continue
			}

			// append the document to the results
			results = append(results, &searchDoc)
		}
	}

	return results, nil
}

func (c *client) GetIndexName() string {
	if c.cfg.IndexPrefix != "" {
		return c.cfg.IndexPrefix + "_global"
	}
	return "global"
}
