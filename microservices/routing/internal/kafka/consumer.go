// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

// Consumer handles consuming messages from Kafka topics
type Consumer struct {
	reader  *kafka.Reader
	logger  zerolog.Logger
	handler MessageHandler
}

// ConsumerConfig contains configuration for Kafka consumer
type ConsumerConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	MinBytes       int
	MaxBytes       int
	MaxWait        time.Duration
	StartOffset    int64
	CommitInterval time.Duration
}

// MessageHandler processes Kafka messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg kafka.Message) error
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config ConsumerConfig, handler MessageHandler, logger zerolog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		MaxWait:        config.MaxWait,
		StartOffset:    config.StartOffset,
		CommitInterval: config.CommitInterval,
		Logger:         kafka.LoggerFunc(logger.Printf),
		ErrorLogger:    kafka.LoggerFunc(logger.Printf),
	})

	return &Consumer{
		reader:  reader,
		logger:  logger,
		handler: handler,
	}
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info().
		Str("topic", c.reader.Config().Topic).
		Str("group_id", c.reader.Config().GroupID).
		Msg("Starting Kafka consumer")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info().Msg("Consumer context cancelled, shutting down")
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return nil
				}
				c.logger.Error().Err(err).Msg("Failed to fetch message")
				continue
			}

			// Process message
			startTime := time.Now()
			if err := c.handler.HandleMessage(ctx, msg); err != nil {
				c.logger.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Failed to handle message")
				// Continue processing other messages even if one fails
				// In production, you might want to send to DLQ
			} else {
				c.logger.Debug().
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Dur("duration", time.Since(startTime)).
					Msg("Successfully processed message")
			}

			// Commit message
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error().Err(err).Msg("Failed to commit message")
			}
		}
	}
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// BatchProcessorHandler handles batch calculation requests
type BatchProcessorHandler struct {
	logger    zerolog.Logger
	processor BatchProcessor
}

// BatchProcessor processes batch route calculations
type BatchProcessor interface {
	ProcessBatch(ctx context.Context, request BatchCalculationRequest) error
}

// NewBatchProcessorHandler creates a new batch processor handler
func NewBatchProcessorHandler(
	processor BatchProcessor,
	logger zerolog.Logger,
) *BatchProcessorHandler {
	return &BatchProcessorHandler{
		logger:    logger,
		processor: processor,
	}
}

// HandleMessage implements MessageHandler for batch processing
func (h *BatchProcessorHandler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	var request BatchCalculationRequest
	if err := json.Unmarshal(msg.Value, &request); err != nil {
		return fmt.Errorf("unmarshaling batch request: %w", err)
	}

	h.logger.Info().
		Str("batch_id", request.BatchID).
		Int("route_count", len(request.Routes)).
		Msg("Processing batch calculation request")

	return h.processor.ProcessBatch(ctx, request)
}

// DataUpdateHandler handles OSM and restriction updates
type DataUpdateHandler struct {
	logger  zerolog.Logger
	updater GraphUpdater
}

// GraphUpdater updates the routing graph
type GraphUpdater interface {
	UpdateOSMData(ctx context.Context, update OSMUpdate) error
	UpdateRestrictions(ctx context.Context, update RestrictionUpdate) error
}

// OSMUpdate represents an OSM data update
type OSMUpdate struct {
	UpdateID     string      `json:"update_id"`
	Timestamp    time.Time   `json:"timestamp"`
	Region       string      `json:"region"`
	BBox         BoundingBox `json:"bbox"`
	NodesAdded   int         `json:"nodes_added"`
	NodesUpdated int         `json:"nodes_updated"`
	NodesDeleted int         `json:"nodes_deleted"`
	EdgesAdded   int         `json:"edges_added"`
	EdgesUpdated int         `json:"edges_updated"`
	EdgesDeleted int         `json:"edges_deleted"`
}

// BoundingBox represents a geographic bounding box
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLon float64 `json:"max_lon"`
}

// RestrictionUpdate represents a truck restriction update
type RestrictionUpdate struct {
	UpdateID        string      `json:"update_id"`
	Timestamp       time.Time   `json:"timestamp"`
	RestrictionType string      `json:"restriction_type"`
	EdgeIDs         []int64     `json:"edge_ids"`
	Restriction     Restriction `json:"restriction"`
}

// Restriction represents a specific restriction
type Restriction struct {
	Type          string    `json:"type"`
	Value         float64   `json:"value"`
	Unit          string    `json:"unit"`
	EffectiveDate time.Time `json:"effective_date"`
	ExpiryDate    time.Time `json:"expiry_date,omitempty"`
}

// NewDataUpdateHandler creates a new data update handler
func NewDataUpdateHandler(updater GraphUpdater, logger zerolog.Logger) *DataUpdateHandler {
	return &DataUpdateHandler{
		logger:  logger,
		updater: updater,
	}
}

// HandleMessage implements MessageHandler for data updates
func (h *DataUpdateHandler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	// Check message type from headers
	var messageType string
	for _, header := range msg.Headers {
		if header.Key == "event_type" {
			messageType = string(header.Value)
			break
		}
	}

	switch messageType {
	case "osm_update":
		var update OSMUpdate
		if err := json.Unmarshal(msg.Value, &update); err != nil {
			return fmt.Errorf("unmarshaling OSM update: %w", err)
		}
		return h.updater.UpdateOSMData(ctx, update)

	case "restriction_update":
		var update RestrictionUpdate
		if err := json.Unmarshal(msg.Value, &update); err != nil {
			return fmt.Errorf("unmarshaling restriction update: %w", err)
		}
		return h.updater.UpdateRestrictions(ctx, update)

	default:
		return fmt.Errorf("unknown message type: %s", messageType)
	}
}
