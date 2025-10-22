package meilisearch

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/zap"
)

type Client struct {
	conn *Connection
}

func NewClient(conn *Connection) *Client {
	return &Client{
		conn: conn,
	}
}

func (c *Client) GetOrCreateIndex(
	ctx context.Context,
	indexName string,
	primaryKey string,
) (*meilisearch.IndexResult, error) {
	index := c.conn.Manager().Index(indexName)
	indexResult, err := index.FetchInfo()

	if err == nil {
		return indexResult, nil
	}

	task, err := c.conn.Manager().CreateIndex(&meilisearch.IndexConfig{
		Uid:        indexName,
		PrimaryKey: primaryKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index %s: %w", indexName, err)
	}

	if err = c.waitForTask(ctx, task.TaskUID); err != nil {
		return nil, fmt.Errorf("failed to wait for index creation: %w", err)
	}

	c.conn.logger.Info("Created new Meilisearch index", zap.String("index", indexName))

	return indexResult, nil
}

func (c *Client) UpdateIndexSettings(
	ctx context.Context,
	indexResult *meilisearch.IndexResult,
	config *meilisearchtype.IndexConfig,
) error {
	settings := &meilisearch.Settings{
		SearchableAttributes: config.SearchableAttributes,
		FilterableAttributes: config.FilterableAttributes,
		SortableAttributes:   config.SortableAttributes,
		DisplayedAttributes:  config.DisplayedAttributes,
		RankingRules:         config.RankingRules,
		StopWords:            config.StopWords,
	}

	task, err := indexResult.UpdateSettings(settings)
	if err != nil {
		return fmt.Errorf("failed to update index settings: %w", err)
	}

	if err = c.waitForTask(ctx, task.TaskUID); err != nil {
		return fmt.Errorf("failed to wait for settings update: %w", err)
	}

	c.conn.logger.Debug("Updated index settings", zap.String("index", indexResult.UID))

	return nil
}

func (c *Client) AddDocuments(
	indexResult *meilisearch.IndexResult,
	documents []*meilisearchtype.SearchDocument,
) (*meilisearch.TaskInfo, error) {
	if len(documents) == 0 {
		return nil, ErrNoDocuments
	}

	task, err := indexResult.AddDocuments(documents, &indexResult.PrimaryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to add documents: %w", err)
	}

	c.conn.logger.Debug("Added documents to index",
		zap.String("index", indexResult.UID),
		zap.Int("count", len(documents)),
		zap.Int64("taskUID", task.TaskUID),
	)

	return &meilisearch.TaskInfo{
		TaskUID:  task.TaskUID,
		IndexUID: task.IndexUID,
		Status:   task.Status,
		Type:     task.Type,
	}, nil
}

func (c *Client) DeleteDocument(
	indexResult *meilisearch.IndexResult,
	documentID string,
) (*meilisearch.TaskInfo, error) {
	task, err := indexResult.DeleteDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	c.conn.logger.Debug("Deleted document from index",
		zap.String("index", indexResult.UID),
		zap.String("documentID", documentID),
		zap.Int64("taskUID", task.TaskUID),
	)

	return &meilisearch.TaskInfo{
		TaskUID:  task.TaskUID,
		IndexUID: task.IndexUID,
		Status:   task.Status,
		Type:     task.Type,
	}, nil
}

func (c *Client) DeleteDocuments(
	indexResult *meilisearch.IndexResult,
	documentIDs []string,
) (*meilisearch.TaskInfo, error) {
	if len(documentIDs) == 0 {
		return nil, ErrNoDocuments
	}

	task, err := indexResult.DeleteDocuments(documentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to delete documents: %w", err)
	}

	c.conn.logger.Debug("Deleted documents from index",
		zap.String("index", indexResult.UID),
		zap.Int("count", len(documentIDs)),
		zap.Int64("taskUID", task.TaskUID),
	)

	return &meilisearch.TaskInfo{
		TaskUID:  task.TaskUID,
		IndexUID: task.IndexUID,
		Status:   task.Status,
		Type:     task.Type,
	}, nil
}

func (c *Client) Search(
	index *meilisearch.IndexResult,
	query string,
	request *meilisearch.SearchRequest,
) (*meilisearch.SearchResponse, error) {
	result, err := index.Search(query, request)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return result, nil
}

func (c *Client) GetTask(taskUID int64) (*meilisearch.Task, error) {
	task, err := c.conn.Manager().GetTask(taskUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

func (c *Client) waitForTask(ctx context.Context, taskUID int64) error {
	timeout := 30 * time.Second
	interval := 100 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		task, err := c.GetTask(taskUID)
		if err != nil {
			return err
		}

		switch task.Status { //nolint:exhaustive // We only need to handle the known statuses
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("task %d failed: %v", taskUID, task.Error)
		case "enqueued", "processing":
			time.Sleep(interval)
			continue
		default:
			return fmt.Errorf("unknown task status: %s", task.Status)
		}
	}

	return fmt.Errorf("task %d timed out after %s", taskUID, timeout)
}
