// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

// Producer handles publishing messages to Kafka topics
type Producer struct {
	writer *kafka.Writer
	logger zerolog.Logger
}

// ProducerConfig contains configuration for Kafka producer
type ProducerConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int
	BatchTimeout time.Duration
	Async        bool
	Compression  string
}

// NewProducer creates a new Kafka producer
func NewProducer(config ProducerConfig, logger zerolog.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		Async:        config.Async,
		RequiredAcks: kafka.RequireOne,
	}

	// Set up compression
	switch config.Compression {
	case "gzip":
		writer.Compression = compress.Gzip
	case "snappy":
		writer.Compression = compress.Snappy
	case "lz4":
		writer.Compression = compress.Lz4
	case "zstd":
		writer.Compression = compress.Zstd
	}

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

// RouteCalculatedEvent represents a route calculation event
type RouteCalculatedEvent struct {
	EventID             string                 `json:"event_id"`
	Timestamp           time.Time              `json:"timestamp"`
	OriginZip           string                 `json:"origin_zip"`
	DestZip             string                 `json:"dest_zip"`
	VehicleType         string                 `json:"vehicle_type"`
	DistanceMiles       float64                `json:"distance_miles"`
	TimeMinutes         float64                `json:"time_minutes"`
	Algorithm           string                 `json:"algorithm"`
	OptimizationType    string                 `json:"optimization_type"`
	ComputeTimeMS       int64                  `json:"compute_time_ms"`
	CacheHit            bool                   `json:"cache_hit"`
	RestrictionsApplied map[string]interface{} `json:"restrictions_applied,omitempty"`
}

// PublishRouteCalculated publishes a route calculation event
func (p *Producer) PublishRouteCalculated(ctx context.Context, event RouteCalculatedEvent) error {
	// Set event ID and timestamp if not provided
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(event.EventID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("route_calculated")},
			{Key: "version", Value: []byte("1.0")},
		},
	}

	// Publish message
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.Error().
			Err(err).
			Str("event_id", event.EventID).
			Str("topic", p.writer.Topic).
			Msg("Failed to publish route calculated event")
		return fmt.Errorf("publishing message: %w", err)
	}

	p.logger.Debug().
		Str("event_id", event.EventID).
		Str("origin", event.OriginZip).
		Str("dest", event.DestZip).
		Msg("Published route calculated event")

	return nil
}

// BatchCalculationRequest represents a batch route calculation request
type BatchCalculationRequest struct {
	BatchID     string         `json:"batch_id"`
	Timestamp   time.Time      `json:"timestamp"`
	CallbackURL string         `json:"callback_url"`
	Routes      []RouteRequest `json:"routes"`
}

// RouteRequest represents a single route in a batch request
type RouteRequest struct {
	ID          string                 `json:"id"`
	OriginZip   string                 `json:"origin_zip"`
	DestZip     string                 `json:"dest_zip"`
	VehicleType string                 `json:"vehicle_type"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

// PublishBatchRequest publishes a batch calculation request
func (p *Producer) PublishBatchRequest(ctx context.Context, request BatchCalculationRequest) error {
	// Set batch ID and timestamp if not provided
	if request.BatchID == "" {
		request.BatchID = uuid.New().String()
	}
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}

	// Marshal request to JSON
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(request.BatchID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("batch_request")},
			{Key: "version", Value: []byte("1.0")},
		},
	}

	// Publish message
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.Error().
			Err(err).
			Str("batch_id", request.BatchID).
			Int("route_count", len(request.Routes)).
			Msg("Failed to publish batch request")
		return fmt.Errorf("publishing message: %w", err)
	}

	p.logger.Info().
		Str("batch_id", request.BatchID).
		Int("route_count", len(request.Routes)).
		Msg("Published batch calculation request")

	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Stats returns producer statistics
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
