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
	key := name + "::" + pattern

	s.mu.RLock()
	tmpl, ok := s.templates[key]
	s.mu.RUnlock()

	if !ok {
		parsed, err := ParseTemplate(pattern)
		if err != nil {
			return "", err
		}

		s.mu.Lock()
		s.templates[key] = parsed
		s.mu.Unlock()
		tmpl = parsed
	}

	return tmpl.Execute(record, primaryKeys)
}

func streamArgs(stream string, payload string) *goredis.XAddArgs {
	return &goredis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"payload": payload,
		},
	}
}
