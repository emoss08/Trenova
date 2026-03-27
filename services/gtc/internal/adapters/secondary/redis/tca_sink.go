package redis

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"go.uber.org/zap"
)

type TCAStreamSink struct {
	*baseSink
}

var _ ports.Sink = (*TCAStreamSink)(nil)

func NewTCAStreamSink(redisURL string, logger *zap.Logger) (*TCAStreamSink, error) {
	base, err := newBaseSink(redisURL, logger.With(zap.String("mode", "tca_stream")))
	if err != nil {
		return nil, err
	}

	return &TCAStreamSink{baseSink: base}, nil
}

func (s *TCAStreamSink) Kind() domain.DestinationKind {
	return domain.DestinationTCAStream
}

func (s *TCAStreamSink) Name() string {
	return "tca_stream"
}

func (s *TCAStreamSink) Initialize(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *TCAStreamSink) Write(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	stream, err := s.renderTemplate(projection.Name, projection.Destination.Stream, projection.PrimaryKeys, record)
	if err != nil {
		s.logger.Error("tca sink: template render failed",
			zap.String("projection", projection.Name),
			zap.Error(err),
		)
		return err
	}

	var changedFields []string
	if record.Operation == domain.OperationUpdate {
		changedFields = domain.ChangedFields(record.OldData, record.NewData)
	}

	payload, err := sonic.Marshal(map[string]any{
		"projection":     projection.Name,
		"operation":      record.Operation,
		"schema":         record.Schema,
		"table":          record.Table,
		"new_data":       record.NewData,
		"old_data":       record.OldData,
		"changed_fields": changedFields,
		"metadata":       record.Metadata,
	})
	if err != nil {
		return fmt.Errorf("marshal tca stream payload: %w", err)
	}

	s.logger.Debug("tca sink: writing event",
		zap.String("stream", stream),
		zap.String("table", record.Table),
		zap.String("operation", string(record.Operation)),
	)

	if err := s.client.XAdd(ctx, streamArgs(stream, string(payload))).Err(); err != nil {
		s.logger.Error("tca sink: xadd failed",
			zap.String("stream", stream),
			zap.Error(err),
		)
		return err
	}

	return nil
}

func (s *TCAStreamSink) HealthCheck(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *TCAStreamSink) Shutdown(ctx context.Context) error {
	return s.client.Close()
}
