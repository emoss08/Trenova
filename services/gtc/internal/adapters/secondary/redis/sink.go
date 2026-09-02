package redis

import (
	"context"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	truncateScanCount   = 512
	truncateDeleteBatch = 256
)

type baseSink struct {
	client    *goredis.Client
	logger    *zap.Logger
	mu        sync.RWMutex
	templates map[string]*Template
}

type JSONSink struct {
	*baseSink
}

type StreamSink struct {
	*baseSink
}

var _ ports.Sink = (*JSONSink)(nil)
var _ ports.Sink = (*StreamSink)(nil)

func NewJSONSink(redisURL string, logger *zap.Logger) (*JSONSink, error) {
	base, err := newBaseSink(redisURL, logger.With(zap.String("mode", "json")))
	if err != nil {
		return nil, err
	}

	return &JSONSink{baseSink: base}, nil
}

func NewStreamSink(redisURL string, logger *zap.Logger) (*StreamSink, error) {
	base, err := newBaseSink(redisURL, logger.With(zap.String("mode", "stream")))
	if err != nil {
		return nil, err
	}

	return &StreamSink{baseSink: base}, nil
}

func newBaseSink(redisURL string, logger *zap.Logger) (*baseSink, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	return &baseSink{
		client:    goredis.NewClient(opts),
		logger:    logger.Named("redis_sink"),
		templates: make(map[string]*Template),
	}, nil
}

func (s *JSONSink) Kind() domain.DestinationKind {
	return domain.DestinationRedisJSON
}

func (s *JSONSink) Name() string {
	return "redis_json"
}

func (s *JSONSink) Initialize(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *JSONSink) Write(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	return s.writeJSON(ctx, projection, record)
}

func (s *JSONSink) HealthCheck(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *JSONSink) Shutdown(ctx context.Context) error {
	return s.client.Close()
}

func (s *JSONSink) writeJSON(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	if record.Operation == domain.OperationTruncate {
		return s.truncateJSON(ctx, projection, record)
	}

	key, err := s.renderTemplate(projection.Name, projection.Destination.KeyTemplate, projection.PrimaryKeys, record)
	if err != nil {
		return err
	}

	if record.Operation == domain.OperationDelete {
		return s.client.Del(ctx, key).Err()
	}

	if record.Operation == domain.OperationUpdate && record.OldData != nil && record.NewData != nil {
		oldOnlyRecord := record
		oldOnlyRecord.NewData = nil

		oldKey, err := s.renderTemplate(projection.Name, projection.Destination.KeyTemplate, projection.PrimaryKeys, oldOnlyRecord)
		if err != nil {
			return err
		}
		if oldKey != "" && oldKey != key {
			if err := s.client.Del(ctx, oldKey).Err(); err != nil {
				return fmt.Errorf("delete old redis json key %s: %w", oldKey, err)
			}
		}
	}

	document, err := domain.SelectFields(record.PrimaryData(), projection.Fields)
	if err != nil {
		return err
	}

	payload, err := sonic.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshal redis json payload: %w", err)
	}

	if err := s.client.Do(ctx, "JSON.SET", key, "$", string(payload)).Err(); err != nil {
		return fmt.Errorf("redis json set %s: %w", key, err)
	}

	return nil
}

func (s *JSONSink) truncateJSON(
	ctx context.Context,
	projection domain.Projection,
	record domain.SourceRecord,
) error {
	tmpl, err := s.template(projection.Name, projection.Destination.KeyTemplate)
	if err != nil {
		return err
	}

	pattern, err := tmpl.WildcardPattern(record, projection.PrimaryKeys)
	if err != nil {
		return fmt.Errorf("truncate projection %s: %w", projection.Name, err)
	}

	deleted, err := s.deleteMatchingKeys(ctx, pattern)
	if err != nil {
		return fmt.Errorf(
			"truncate projection %s: delete keys matching %q: %w",
			projection.Name,
			pattern,
			err,
		)
	}

	s.logger.Info("truncated redis json projection",
		zap.String("projection", projection.Name),
		zap.String("table", record.FullTableName()),
		zap.String("pattern", pattern),
		zap.Int64("deleted_keys", deleted),
	)

	return nil
}

func (s *baseSink) deleteMatchingKeys(ctx context.Context, pattern string) (int64, error) {
	var deleted int64
	batch := make([]string, 0, truncateDeleteBatch)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		removed, err := s.client.Unlink(ctx, batch...).Result()
		if err != nil {
			return err
		}
		deleted += removed
		batch = batch[:0]
		return nil
	}

	iter := s.client.Scan(ctx, 0, pattern, truncateScanCount).Iterator()
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= truncateDeleteBatch {
			if err := flush(); err != nil {
				return deleted, err
			}
		}
	}
	if err := iter.Err(); err != nil {
		return deleted, err
	}
	if err := flush(); err != nil {
		return deleted, err
	}

	return deleted, nil
}

func (s *StreamSink) Kind() domain.DestinationKind {
	return domain.DestinationRedisStream
}

func (s *StreamSink) Name() string {
	return "redis_stream"
}

func (s *StreamSink) Initialize(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *StreamSink) Write(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	return s.writeStream(ctx, projection, record)
}

func (s *StreamSink) HealthCheck(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *StreamSink) Shutdown(ctx context.Context) error {
	return s.client.Close()
}

func (s *StreamSink) writeStream(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	stream, err := s.renderTemplate(projection.Name, projection.Destination.Stream, projection.PrimaryKeys, record)
	if err != nil {
		return err
	}

	payload, err := sonic.Marshal(map[string]any{
		"projection": projection.Name,
		"operation":  record.Operation,
		"schema":     record.Schema,
		"table":      record.Table,
		"new_data":   record.NewData,
		"old_data":   record.OldData,
		"metadata":   record.Metadata,
	})
	if err != nil {
		return fmt.Errorf("marshal redis stream payload: %w", err)
	}

	return s.client.XAdd(ctx, streamArgs(stream, string(payload))).Err()
}

func (s *baseSink) renderTemplate(name string, pattern string, primaryKeys []string, record domain.SourceRecord) (string, error) {
	tmpl, err := s.template(name, pattern)
	if err != nil {
		return "", err
	}

	return tmpl.Execute(record, primaryKeys)
}

func (s *baseSink) template(name string, pattern string) (*Template, error) {
	key := name + "::" + pattern

	s.mu.RLock()
	tmpl, ok := s.templates[key]
	s.mu.RUnlock()

	if ok {
		return tmpl, nil
	}

	parsed, err := ParseTemplate(pattern)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.templates[key] = parsed
	s.mu.Unlock()

	return parsed, nil
}

func streamArgs(stream string, payload string) *goredis.XAddArgs {
	return &goredis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"payload": payload,
		},
	}
}
