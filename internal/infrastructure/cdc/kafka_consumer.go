package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

type KafkaConsumerParams struct {
	fx.In

	Logger *logger.Logger
	Config *config.Manager
}

type KafkaConsumerService struct {
	l        *zerolog.Logger
	config   *config.KafkaConfig
	reader   *kafka.Reader
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex
	handlers map[string]services.CDCEventHandler // table -> handler mapping
}

// topicInfo holds information parsed from a Kafka topic
type topicInfo struct {
	database string
	schema   string
	table    string
}

// DebeziumChangeEvent represents the structure of Debezium change events
type DebeziumChangeEvent struct {
	Schema  DebeziumSchema  `json:"schema"`
	Payload DebeziumPayload `json:"payload"`
}

type DebeziumSchema struct {
	Type   string `json:"type"`
	Fields []struct {
		Field string `json:"field"`
		Type  string `json:"type"`
	} `json:"fields"`
}

type DebeziumPayload struct {
	Before      *json.RawMessage `json:"before"`
	After       *json.RawMessage `json:"after"`
	Source      DebeziumSource   `json:"source"`
	Operation   string           `json:"op"` // c=create, u=update, d=delete, r=read
	Timestamp   int64            `json:"ts_ms"`
	Transaction *struct {
		ID string `json:"id"`
	} `json:"transaction,omitempty"`
}

type DebeziumSource struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Name      string `json:"name"`
	Timestamp int64  `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	Database  string `json:"db"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Sequence  string `json:"sequence,omitempty"`
	LSN       int64  `json:"lsn,omitempty"`
	TXID      int64  `json:"txId,omitempty"`
}

func NewKafkaConsumerService(p KafkaConsumerParams) services.CDCService {
	log := p.Logger.With().
		Str("service", "kafka-consumer").
		Logger()

	return &KafkaConsumerService{
		l:        &log,
		config:   p.Config.Kafka(),
		handlers: make(map[string]services.CDCEventHandler),
	}
}

func (s *KafkaConsumerService) RegisterHandler(table string, handler services.CDCEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[table] = handler
	s.l.Info().
		Str("table", table).
		Str("handler_type", fmt.Sprintf("%T", handler)).
		Msg("Registered CDC handler for table")
}

func (s *KafkaConsumerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("kafka consumer already running")
	}

	if !s.config.Enabled {
		s.l.Info().Msg("Kafka consumer disabled via configuration")
		return nil
	}

	s.l.Info().
		Strs("brokers", s.config.Brokers).
		Str("topic_pattern", s.config.TopicPattern).
		Str("group_id", s.config.ConsumerGroupID).
		Msg("🚀 Starting Kafka consumer service")

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Parse start offset
	startOffset := kafka.LastOffset
	if strings.ToLower(s.config.StartOffset) == "earliest" {
		startOffset = kafka.FirstOffset
	}

	// Get topics that match our pattern
	topics, err := s.getMatchingTopics()
	if err != nil {
		s.l.Warn().Err(err).Msg("Failed to get matching topics, will retry later")
		// Start with common topics based on pattern
		topics = s.getDefaultTopics()
	}

	if len(topics) == 0 {
		s.l.Warn().Msg("No topics found, starting consumer anyway to listen for new topics")
		topics = s.getDefaultTopics()
	}

	s.l.Info().Strs("topics", topics).Msg("Subscribing to topics")

	// Create Kafka reader for the first topic (we'll handle multiple topics differently)
	if len(topics) > 0 {
		s.reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        s.config.Brokers,
			Topic:          topics[0], // For now, start with first topic
			GroupID:        s.config.ConsumerGroupID,
			StartOffset:    startOffset,
			MinBytes:       1,
			MaxBytes:       10e6, // 10MB
			CommitInterval: s.config.CommitInterval,
			MaxWait:        1 * time.Second,
		})
	}

	s.running = true

	// Start consumer goroutines for all topics
	for _, topic := range topics {
		s.wg.Add(1)
		go s.consumeFromTopic(topic, startOffset)
	}

	s.l.Info().Msg("Kafka consumer service started successfully")
	return nil
}

func (s *KafkaConsumerService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.l.Info().Msg("Stopping Kafka consumer service")

	if s.cancel != nil {
		s.cancel()
	}

	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			s.l.Error().Err(err).Msg("Error closing Kafka reader")
		}
	}

	s.wg.Wait()
	s.running = false

	s.l.Info().Msg("Kafka consumer service stopped")
	return nil
}

func (s *KafkaConsumerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *KafkaConsumerService) GetMetrics() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := map[string]any{
		"running": s.running,
		"config": map[string]any{
			"brokers":           s.config.Brokers,
			"consumer_group_id": s.config.ConsumerGroupID,
			"topic_pattern":     s.config.TopicPattern,
			"enabled":           s.config.Enabled,
		},
	}

	if s.reader != nil {
		stats := s.reader.Stats()
		metrics["kafka_stats"] = map[string]any{
			"messages":   stats.Messages,
			"bytes":      stats.Bytes,
			"rebalances": stats.Rebalances,
			"timeouts":   stats.Timeouts,
			"errors":     stats.Errors,
		}
	}

	return metrics
}

// getMatchingTopics discovers topics that match the configured pattern
func (s *KafkaConsumerService) getMatchingTopics() ([]string, error) {
	conn, err := kafka.Dial("tcp", s.config.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("failed to read partitions: %w", err)
	}

	topicSet := make(map[string]bool)
	pattern := s.convertPatternToRegex(s.config.TopicPattern)

	for _, partition := range partitions {
		if pattern.MatchString(partition.Topic) {
			topicSet[partition.Topic] = true
		}
	}

	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics, nil
}

// convertPatternToRegex converts a glob pattern to regex
func (s *KafkaConsumerService) convertPatternToRegex(pattern string) *regexp.Regexp {
	// Convert glob pattern to regex
	// Replace * with .*
	regexPattern := strings.ReplaceAll(pattern, "*", ".*")
	regexPattern = "^" + regexPattern + "$"

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		s.l.Warn().Err(err).Str("pattern", pattern).Msg("Invalid pattern, using match-all")
		return regexp.MustCompile(".*")
	}

	return regex
}

// getDefaultTopics returns a list of default topics based on the pattern
func (s *KafkaConsumerService) getDefaultTopics() []string {
	// Extract base from pattern and create common table topics
	if strings.Contains(s.config.TopicPattern, "trenova.public") {
		return []string{
			"trenova.public.shipments",
			"trenova.public.users",
			"trenova.public.customers",
			"trenova.public.equipment",
			"trenova.public.locations",
		}
	}
	return []string{}
}

// consumeFromTopic consumes messages from a specific topic
func (s *KafkaConsumerService) consumeFromTopic(topic string, startOffset int64) {
	defer s.wg.Done()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        s.config.Brokers,
		Topic:          topic,
		GroupID:        s.config.ConsumerGroupID,
		StartOffset:    startOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: s.config.CommitInterval,
		MaxWait:        500 * time.Millisecond, // Shorter wait to reduce timeout errors
	})
	defer reader.Close()

	s.l.Info().Str("topic", topic).Msg("Started consuming from topic")

	for {
		select {
		case <-s.ctx.Done():
			s.l.Info().Str("topic", topic).Msg("Consumer context cancelled, stopping consumption")
			return
		default:
			message, err := reader.ReadMessage(s.ctx)
			if err != nil {
				if err == context.Canceled {
					s.l.Info().Str("topic", topic).Msg("Consumer context cancelled during read")
					return
				}
				// Check if it's a timeout error (which is normal when no messages)
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					s.l.Debug().Str("topic", topic).Msg("Read timeout - no new messages (normal)")
					continue
				}
				// Log other errors as warnings instead of errors
				s.l.Warn().Err(err).Str("topic", topic).Msg("Kafka read issue")
				time.Sleep(1 * time.Second) // Brief pause before retrying
				continue
			}

			if err := s.processMessage(message); err != nil {
				s.l.Error().
					Err(err).
					Str("topic", message.Topic).
					Int("partition", message.Partition).
					Int64("offset", message.Offset).
					Msg("Error processing Kafka message")
			}
		}
	}
}

func (s *KafkaConsumerService) processMessage(message kafka.Message) error {
	s.l.Debug().
		Str("topic", message.Topic).
		Int("partition", message.Partition).
		Int64("offset", message.Offset).
		Msg("Processing Kafka message")

	// Parse Debezium change event
	var changeEvent DebeziumChangeEvent
	if err := json.Unmarshal(message.Value, &changeEvent); err != nil {
		return fmt.Errorf("failed to unmarshal change event: %w", err)
	}

	// Extract table information
	tableName := changeEvent.Payload.Source.Table

	// Check if we have a handler for this table
	s.mu.RLock()
	handler, exists := s.handlers[tableName]
	s.mu.RUnlock()

	if !exists {
		s.l.Debug().
			Str("table", tableName).
			Msg("No handler registered for table, skipping")
		return nil
	}

	// Convert Debezium event to CDC event
	cdcEvent, err := s.convertToCDCEvent(changeEvent)
	if err != nil {
		return fmt.Errorf("failed to convert to CDC event: %w", err)
	}

	// Route to appropriate handler
	if err := handler.HandleEvent(*cdcEvent); err != nil {
		return fmt.Errorf("handler failed for table %s: %w", tableName, err)
	}

	s.l.Debug().
		Str("table", tableName).
		Str("operation", cdcEvent.Operation).
		Msg("Successfully processed CDC event")

	return nil
}

// convertToCDCEvent converts a Debezium event to our generic CDC event format
func (s *KafkaConsumerService) convertToCDCEvent(
	debeziumEvent DebeziumChangeEvent,
) (*services.CDCEvent, error) {
	var before, after map[string]any

	// Parse before state
	if debeziumEvent.Payload.Before != nil {
		if err := json.Unmarshal(*debeziumEvent.Payload.Before, &before); err != nil {
			return nil, fmt.Errorf("failed to unmarshal before state: %w", err)
		}
	}

	// Parse after state
	if debeziumEvent.Payload.After != nil {
		if err := json.Unmarshal(*debeziumEvent.Payload.After, &after); err != nil {
			return nil, fmt.Errorf("failed to unmarshal after state: %w", err)
		}
	}

	// Map operation
	var operation string
	switch debeziumEvent.Payload.Operation {
	case "c":
		operation = "create"
	case "u":
		operation = "update"
	case "d":
		operation = "delete"
	case "r":
		operation = "read"
	default:
		operation = debeziumEvent.Payload.Operation
	}

	return &services.CDCEvent{
		Operation: operation,
		Table:     debeziumEvent.Payload.Source.Table,
		Schema:    debeziumEvent.Payload.Source.Schema,
		Before:    before,
		After:     after,
		Metadata: services.CDCMetadata{
			Timestamp: debeziumEvent.Payload.Timestamp,
			Source: services.CDCSource{
				Database:  debeziumEvent.Payload.Source.Database,
				Schema:    debeziumEvent.Payload.Source.Schema,
				Table:     debeziumEvent.Payload.Source.Table,
				Connector: debeziumEvent.Payload.Source.Connector,
				Version:   debeziumEvent.Payload.Source.Version,
			},
			LSN: debeziumEvent.Payload.Source.LSN,
		},
	}, nil
}
