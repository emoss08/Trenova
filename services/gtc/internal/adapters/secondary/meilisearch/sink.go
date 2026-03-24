package meilisearch

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/zap"
)

type Sink struct {
	client           meilisearch.ServiceManager
	logger           *zap.Logger
	searchableConfig sync.Map
	filterableConfig sync.Map
}

var _ ports.Sink = (*Sink)(nil)

func NewSink(url string, apiKey string, logger *zap.Logger) *Sink {
	return &Sink{
		client: meilisearch.New(url, meilisearch.WithAPIKey(apiKey)),
		logger: logger.Named("meilisearch_sink"),
	}
}

func (s *Sink) Kind() domain.DestinationKind {
	return domain.DestinationMeilisearch
}

func (s *Sink) Name() string {
	return "meilisearch"
}

func (s *Sink) Initialize(ctx context.Context) error {
	_, err := s.client.Health()
	return err
}

func (s *Sink) Write(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	index := s.client.Index(projection.Destination.Index)
	if err := s.ensureSearchableAttributes(ctx, index, projection); err != nil {
		return err
	}
	if err := s.ensureFilterableAttributes(ctx, index, projection); err != nil {
		return err
	}

	keyField, key, err := documentKey(record, projection.PrimaryKeys)
	if err != nil {
		return err
	}

	if record.Operation == domain.OperationDelete {
		task, err := index.DeleteDocument(key, nil)
		if err != nil {
			return err
		}
		return s.waitForTask(ctx, projection, record, task, "delete document")
	}

	document, err := domain.SelectFields(record.PrimaryData(), projection.Fields)
	if err != nil {
		return err
	}

	if keyField == "_pk" {
		document["_pk"] = key
	}
	document["_projection"] = projection.Name
	document["_source_table"] = projection.FullTableName()

	task, err := index.AddDocuments([]map[string]any{document}, &meilisearch.DocumentOptions{
		PrimaryKey: meilisearch.StringPtr(keyField),
	})
	if err != nil {
		return err
	}

	return s.waitForTask(ctx, projection, record, task, "add document")
}

func (s *Sink) HealthCheck(ctx context.Context) error {
	_, err := s.client.Health()
	return err
}

func (s *Sink) Shutdown(ctx context.Context) error {
	s.client.Close()
	return nil
}

func (s *Sink) ensureSearchableAttributes(
	ctx context.Context,
	index meilisearch.IndexManager,
	projection domain.Projection,
) error {
	if len(projection.SearchableFields) == 0 {
		return nil
	}

	if current, loaded := s.searchableConfig.LoadOrStore(
		projection.Destination.Index,
		append([]string(nil), projection.SearchableFields...),
	); loaded {
		existing, _ := current.([]string)
		if !slices.Equal(existing, projection.SearchableFields) {
			return fmt.Errorf(
				"conflicting searchable fields for index %s: %v != %v",
				projection.Destination.Index,
				existing,
				projection.SearchableFields,
			)
		}
		return nil
	}

	task, err := index.UpdateSearchableAttributes(&projection.SearchableFields)
	if err != nil {
		s.searchableConfig.Delete(projection.Destination.Index)
		return fmt.Errorf("update searchable attributes for %s: %w", projection.Name, err)
	}

	if err := s.waitForTask(ctx, projection, domain.SourceRecord{}, task, "update searchable attributes"); err != nil {
		s.searchableConfig.Delete(projection.Destination.Index)
		return err
	}

	return nil
}

func (s *Sink) ensureFilterableAttributes(
	ctx context.Context,
	index meilisearch.IndexManager,
	projection domain.Projection,
) error {
	if len(projection.FilterableFields) == 0 {
		return nil
	}

	filterableFields := append([]string(nil), projection.FilterableFields...)
	filterableSettings := make([]any, 0, len(filterableFields))
	for _, field := range filterableFields {
		filterableSettings = append(filterableSettings, field)
	}

	if current, loaded := s.filterableConfig.LoadOrStore(
		projection.Destination.Index,
		append([]string(nil), filterableFields...),
	); loaded {
		existing, _ := current.([]string)
		if !slices.Equal(existing, filterableFields) {
			return fmt.Errorf(
				"conflicting filterable fields for index %s: %v != %v",
				projection.Destination.Index,
				existing,
				filterableFields,
			)
		}
		return nil
	}

	task, err := index.UpdateFilterableAttributes(&filterableSettings)
	if err != nil {
		s.filterableConfig.Delete(projection.Destination.Index)
		return fmt.Errorf("update filterable attributes for %s: %w", projection.Name, err)
	}

	if err := s.waitForTask(ctx, projection, domain.SourceRecord{}, task, "update filterable attributes"); err != nil {
		s.filterableConfig.Delete(projection.Destination.Index)
		return err
	}

	return nil
}

func primaryKey(record domain.SourceRecord, keyFields []string) (string, error) {
	values, err := domain.PrimaryKey(record, keyFields)
	if err != nil {
		return "", fmt.Errorf("record for %s: %w", record.FullTableName(), err)
	}
	return domain.KeyString(values), nil
}

func documentKey(record domain.SourceRecord, keyFields []string) (string, string, error) {
	data := record.PrimaryData()
	if data != nil {
		if rawID, ok := data["id"]; ok {
			id := fmt.Sprintf("%v", rawID)
			if id != "" && id != "<nil>" {
				return "id", id, nil
			}
		}
	}

	key, err := primaryKey(record, keyFields)
	if err != nil {
		return "", "", err
	}

	return "_pk", key, nil
}

func (s *Sink) waitForTask(
	ctx context.Context,
	projection domain.Projection,
	record domain.SourceRecord,
	taskInfo *meilisearch.TaskInfo,
	action string,
) error {
	if taskInfo == nil {
		return fmt.Errorf("%s for projection %s returned no task info", action, projection.Name)
	}

	task, err := s.client.WaitForTaskWithContext(ctx, taskInfo.TaskUID, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("%s for projection %s: wait for task %d: %w", action, projection.Name, taskInfo.TaskUID, err)
	}

	if task.Status == meilisearch.TaskStatusFailed {
		s.logger.Error("meilisearch task failed",
			zap.String("projection", projection.Name),
			zap.String("index", projection.Destination.Index),
			zap.String("action", action),
			zap.Int64("task_uid", task.UID),
			zap.String("task_type", string(task.Type)),
			zap.String("operation", record.Operation.String()),
			zap.String("table", record.FullTableName()),
			zap.String("record_key", primaryKeyForLog(record, projection.PrimaryKeys)),
			zap.String("error_code", task.Error.Code),
			zap.String("error_type", task.Error.Type),
			zap.String("error_message", task.Error.Message),
		)
		return fmt.Errorf(
			"%s for projection %s failed: %s (%s)",
			action,
			projection.Name,
			task.Error.Message,
			task.Error.Code,
		)
	}

	return nil
}

func primaryKeyForLog(record domain.SourceRecord, keyFields []string) string {
	_, key, err := documentKey(record, keyFields)
	if err != nil {
		return ""
	}

	return key
}
