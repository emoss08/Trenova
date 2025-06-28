// Package cdc implements Change Data Capture (CDC) functionality for real-time database event processing.
// It provides a Kafka-based consumer service that listens to Debezium change events and routes them
// to appropriate table-specific handlers for processing.
package cdc

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/cdctypes"
	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/segmentio/kafka-go"
	"github.com/sourcegraph/conc"
	"go.uber.org/fx"
)

type KafkaConsumerParams struct {
	fx.In

	Logger *logger.Logger
	Config *config.Manager
}

type KafkaConsumerService struct {
	l            *zerolog.Logger
	config       *config.KafkaConfig
	reader       *kafka.Reader
	ctx          context.Context
	cancel       context.CancelFunc
	wg           *conc.WaitGroup
	running      bool
	mu           sync.RWMutex
	handlers     map[string]services.CDCEventHandler // * table -> handler mapping
	schemaClient *srclient.SchemaRegistryClient
	avroSchemas  map[string]*goavro.Codec // * subject -> codec mapping
	schemasMu    sync.RWMutex
}

// NewKafkaConsumerService initializes a new Kafka CDC consumer service with its dependencies.
// This service manages the consumption of Debezium change events from Kafka topics and routes
// them to registered table-specific handlers for processing.
//
// Parameters:
//   - p: KafkaConsumerParams containing dependencies (logger, config).
//
// Returns:
//   - services.CDCService: A ready-to-use CDC consumer service instance.
func NewKafkaConsumerService(p KafkaConsumerParams) services.CDCService {
	cfg := p.Config.Kafka()

	log := p.Logger.With().
		Str("service", "kafka-consumer").
		Logger()

	// Initialize schema registry client
	schemaClient := srclient.CreateSchemaRegistryClient(cfg.SchemaRegistryURL)

	return &KafkaConsumerService{
		l:            &log,
		config:       p.Config.Kafka(),
		wg:           conc.NewWaitGroup(),
		handlers:     make(map[string]services.CDCEventHandler),
		schemaClient: schemaClient,
		avroSchemas:  make(map[string]*goavro.Codec),
		schemasMu:    sync.RWMutex{},
	}
}

// RegisterHandler registers a table-specific handler for processing CDC events.
// Each table can have one handler that will process all change events (CREATE, UPDATE, DELETE)
// for that specific table.
//
// Parameters:
//   - table: The database table name to handle events for.
//   - handler: The handler implementation that will process events for this table.
func (s *KafkaConsumerService) RegisterHandler(table string, handler services.CDCEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[table] = handler
	s.l.Info().
		Str("table", table).
		Str("handler_type", fmt.Sprintf("%T", handler)).
		Msg("Registered CDC handler for table")
}

// Start initializes and starts the Kafka CDC consumer service.
// It discovers available topics, creates readers for each topic matching the configured pattern,
// and launches goroutines to consume messages from each topic.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//
// Returns:
//   - error: If the service fails to start or is already running.
func (s *KafkaConsumerService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return cdctypes.ErrConsumerAlreadyRunning
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

	// * Create a background context for long-running operations instead of using the startup context
	// * The startup context has a timeout and will cancel after application starts
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// * Parse start offset
	startOffset := kafka.LastOffset
	if strings.EqualFold(s.config.StartOffset, "earliest") {
		startOffset = kafka.FirstOffset
	}

	// * Get topics that match our pattern
	topics, err := s.getMatchingTopics()
	if err != nil {
		s.l.Warn().Err(err).Msg("Failed to get matching topics, will retry later")
		// * Start with common topics based on pattern
		topics = s.getDefaultTopics()
	}

	if len(topics) == 0 {
		s.l.Warn().Msg("No topics found, starting consumer anyway to listen for new topics")
		topics = s.getDefaultTopics()
	}

	s.l.Info().Strs("topics", topics).Msg("Subscribing to topics")

	if len(topics) > 0 {
		s.reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers:                s.config.Brokers,
			Topic:                  topics[0], // For now, start with first topic
			GroupID:                s.config.ConsumerGroupID,
			StartOffset:            startOffset,
			MinBytes:               1,
			MaxBytes:               10e6, // 10MB
			CommitInterval:         s.config.CommitInterval,
			MaxWait:                1 * time.Second,
			ReadBatchTimeout:       10 * time.Second,
			QueueCapacity:          100,
			SessionTimeout:         30 * time.Second,
			RebalanceTimeout:       30 * time.Second,
			PartitionWatchInterval: 5 * time.Second,
			ErrorLogger:            kafka.LoggerFunc(s.logKafkaError),
		})
	}

	s.running = true

	// Start consumer goroutines for all topics
	for _, topic := range topics {
		s.wg.Go(func() {
			s.consumeFromTopic(topic, startOffset)
		})
	}

	s.l.Info().Msg("Kafka consumer service started successfully")
	return nil
}

// Stop gracefully shuts down the Kafka CDC consumer service.
// It cancels the context to signal all consumers to stop, closes the Kafka reader,
// and waits for all goroutines to complete.
//
// Returns:
//   - error: If there are issues during shutdown (logged but not returned).
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

// IsRunning returns the current running state of the CDC consumer service.
// Thread-safe method that can be called concurrently.
//
// Returns:
//   - bool: True if the service is currently running, false otherwise.
func (s *KafkaConsumerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetMetrics returns runtime metrics and statistics about the CDC consumer service.
// Includes configuration details, running status, and Kafka reader statistics.
//
// Returns:
//   - map[string]any: Metrics data including config, status, and Kafka stats.
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
		"handlers_registered": len(s.handlers),
		"schemas_cached":      len(s.avroSchemas),
	}

	if s.reader != nil {
		stats := s.reader.Stats()
		metrics["kafka_stats"] = map[string]any{
			"messages":       stats.Messages,
			"bytes":          stats.Bytes,
			"rebalances":     stats.Rebalances,
			"timeouts":       stats.Timeouts,
			"errors":         stats.Errors,
			"dials":          stats.Dials,
			"fetches":        stats.Fetches,
			"offset":         stats.Offset,
			"lag":            stats.Lag,
			"min_bytes":      stats.MinBytes,
			"max_bytes":      stats.MaxBytes,
			"queue_capacity": stats.QueueCapacity,
			"queue_length":   stats.QueueLength,
		}
	}

	return metrics
}

// logKafkaError handles Kafka error logging in a structured way
func (s *KafkaConsumerService) logKafkaError(msg string, args ...any) {
	// * Downgrade rebalancing messages to debug level as they're normal operation
	if strings.Contains(msg, "Rebalance In Progress") || strings.Contains(msg, "[27]") {
		s.l.Debug().Msgf("Kafka rebalancing: "+msg, args...)
		return
	}
	// * Downgrade timeout errors to debug level as they're normal when no messages
	if strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "timeout") {
		s.l.Debug().Msgf("Kafka timeout (normal when idle): "+msg, args...)
		return
	}
	s.l.Error().Msgf("Kafka error: "+msg, args...)
}

// getMatchingTopics discovers Kafka topics that match the configured pattern.
// It connects to Kafka, reads available partitions, and filters topics using regex matching.
//
// Returns:
//   - []string: List of topic names that match the configured pattern.
//   - error: If Kafka connection or topic discovery fails.
func (s *KafkaConsumerService) getMatchingTopics() ([]string, error) {
	conn, err := kafka.Dial("tcp", s.config.Brokers[0])
	if err != nil {
		return nil, oops.
			In("kafka_consumer").
			With("broker", s.config.Brokers[0]).
			Time(time.Now()).
			Wrapf(err, "failed to connect to Kafka")
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, oops.
			In("kafka_consumer").
			With("broker", s.config.Brokers[0]).
			Time(time.Now()).
			Wrapf(err, "failed to read partitions")
	}

	topicSet := make(map[string]bool)
	pattern := s.convertPatternToRegex(s.config.TopicPattern)

	for i := range partitions {
		if pattern.MatchString(partitions[i].Topic) {
			topicSet[partitions[i].Topic] = true
		}
	}

	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics, nil
}

// convertPatternToRegex converts a glob-style pattern to a compiled regular expression.
// Supports wildcard (*) patterns commonly used in topic naming conventions.
//
// Parameters:
//   - pattern: Glob pattern string (e.g., "trenova.public.*")
//
// Returns:
//   - *regexp.Regexp: Compiled regex pattern, or match-all pattern if compilation fails.
func (s *KafkaConsumerService) convertPatternToRegex(pattern string) *regexp.Regexp {
	// * Convert glob pattern to regex
	// * Replace * with .*
	regexPattern := strings.ReplaceAll(pattern, "*", ".*")
	regexPattern = "^" + regexPattern + "$"

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		s.l.Warn().Err(err).Str("pattern", pattern).Msg("Invalid pattern, using match-all")
		return regexp.MustCompile(".*")
	}

	return regex
}

// getDefaultTopics provides a fallback list of common topics when topic discovery fails.
// Returns a predefined set of table topics based on the configured pattern to ensure
// the consumer can start even if Kafka metadata is temporarily unavailable.
//
// Returns:
//   - []string: List of default topic names for common database tables.
func (s *KafkaConsumerService) getDefaultTopics() []string {
	// * Extract base from pattern and create common table topics
	if strings.Contains(s.config.TopicPattern, "trenova.public") {
		return []string{
			"trenova.public.shipments",
			"trenova.public.shipment_moves",
			"trenova.public.stops",
			"trenova.public.assignments",
			"trenova.public.users",
			"trenova.public.workers",
			"trenova.public.customers",
			"trenova.public.tractors",
			"trenova.public.trailers",
			"trenova.public.locations",
		}
	}
	return []string{}
}

// newTopicReader creates and configures a new Kafka reader for a specific topic.
func (s *KafkaConsumerService) newTopicReader(topic string, startOffset int64) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:                s.config.Brokers,
		Topic:                  topic,
		GroupID:                s.config.ConsumerGroupID,
		StartOffset:            startOffset,
		MinBytes:               1,
		MaxBytes:               10e6,
		CommitInterval:         s.config.CommitInterval,
		MaxWait:                500 * time.Millisecond,
		ReadBatchTimeout:       10 * time.Second,
		QueueCapacity:          100,
		SessionTimeout:         30 * time.Second,
		RebalanceTimeout:       30 * time.Second,
		PartitionWatchInterval: 5 * time.Second,
		ErrorLogger:            kafka.LoggerFunc(s.logKafkaError),
	})
}

// handleReadError centralizes error handling for Kafka message reads.
// It implements an exponential backoff strategy for retriable errors.
// Returns false if consumption should stop.
func (s *KafkaConsumerService) handleReadError(
	err error,
	topic string,
	backoff *time.Duration,
) bool {
	if eris.Is(err, context.Canceled) {
		s.l.Info().Str("topic", topic).Msg("Consumer context cancelled during read")
		return false
	}

	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		s.l.Debug().Str("topic", topic).Msg("Read timeout - no new messages (normal)")
		*backoff = 0 // Reset backoff on normal timeout
		return true
	}

	// Implement exponential backoff for other errors
	const maxBackoff = 30 * time.Second
	if *backoff == 0 {
		*backoff = 1 * time.Second
	} else {
		*backoff *= 2
		if *backoff > maxBackoff {
			*backoff = maxBackoff
		}
	}
	s.l.Warn().
		Err(err).
		Str("topic", topic).
		Dur("backoff", *backoff).
		Msg("Kafka read error, backing off")
	time.Sleep(*backoff)

	return true
}

// processAndLogMessage wraps the message processing and error logging.
func (s *KafkaConsumerService) processAndLogMessage(message *kafka.Message) {
	processCtx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	_ = processCtx // Will be used in future for passing context to handlers

	if err := s.processMessage(message); err != nil {
		s.l.Error().
			Err(err).
			Str("topic", message.Topic).
			Int("partition", message.Partition).
			Int64("offset", message.Offset).
			Bytes("key", message.Key).
			Int("value_size", len(message.Value)).
			Msg("Error processing Kafka message")
	}
}

// consumeFromTopic continuously consumes and processes messages from a specific Kafka topic.
// This method runs in its own goroutine and handles message reading, error handling,
// and graceful shutdown when the context is cancelled.
//
// Parameters:
//   - topic: The Kafka topic name to consume from.
//   - startOffset: Starting offset position (FirstOffset or LastOffset).
func (s *KafkaConsumerService) consumeFromTopic(topic string, startOffset int64) {
	reader := s.newTopicReader(topic, startOffset)
	defer reader.Close()

	s.l.Info().Str("topic", topic).Msg("Started consuming from topic")

	var backoff time.Duration

	for {
		select {
		case <-s.ctx.Done():
			s.l.Info().Str("topic", topic).Msg("Consumer context cancelled, stopping consumption")
			return
		default:
			message, err := reader.ReadMessage(s.ctx)
			if err != nil {
				if !s.handleReadError(err, topic, &backoff) {
					return
				}
				continue
			}

			// Reset backoff on successful read
			backoff = 0
			s.processAndLogMessage(&message)
		}
	}
}

// decodeAvroMessage decodes an Avro-encoded Kafka message.
func (s *KafkaConsumerService) decodeAvroMessage(message *kafka.Message) (map[string]any, error) {
	// ! Kafka Connect Avro messages have the format: [magic_byte][schema_id][avro_data]
	// ! The first byte is 0x0 (magic byte), followed by 4 bytes for schema ID
	if len(message.Value) < 5 {
		return nil, oops.
			In("kafka_consumer").
			With("topic", message.Topic).
			Time(time.Now()).
			New("message too short for Avro format")
	}

	// * Extract schema ID (bytes 1-4, big endian)
	schemaID := int(
		message.Value[1],
	)<<24 | int(
		message.Value[2],
	)<<16 | int(
		message.Value[3],
	)<<8 | int(
		message.Value[4],
	)

	// * Get schema from registry using schema ID
	schema, err := s.schemaClient.GetSchema(schemaID)
	if err != nil {
		return nil, oops.
			In("kafka_consumer").
			With("schema_id", schemaID).
			With("topic", message.Topic).
			Time(time.Now()).
			Wrapf(err, "failed to get schema by ID")
	}

	// * Create codec if not cached
	codec, err := goavro.NewCodec(schema.Schema())
	if err != nil {
		return nil, oops.
			In("kafka_consumer").
			With("schema_id", schemaID).
			Time(time.Now()).
			Wrapf(err, "failed to create Avro codec")
	}

	// * Decode the Avro data (skip first 5 bytes which are magic byte + schema ID)
	native, _, err := codec.NativeFromBinary(message.Value[5:])
	if err != nil {
		return nil, oops.
			In("kafka_consumer").
			With("schema_id", schemaID).
			With("topic", message.Topic).
			Time(time.Now()).
			Wrapf(err, "failed to decode Avro message")
	}

	// * Convert to map[string]any
	if result, ok := native.(map[string]any); ok {
		return result, nil
	}

	return nil, oops.
		In("kafka_consumer").
		With("schema_id", schemaID).
		With("topic", message.Topic).
		Time(time.Now()).
		New("decoded Avro message is not a map")
}

// processMessage handles individual Kafka messages containing Debezium change events.
// It unmarshals the JSON payload, extracts table information, finds the appropriate handler,
// and routes the event for processing.
//
// Parameters:
//   - message: The Kafka message containing the Debezium change event.
//
// Returns:
//   - error: If message processing fails at any stage.
func (s *KafkaConsumerService) processMessage(message *kafka.Message) error {
	s.l.Debug().
		Str("topic", message.Topic).
		Int("partition", message.Partition).
		Int64("offset", message.Offset).
		Msg("Processing Kafka message")

	// * Decode Avro message
	avroData, err := s.decodeAvroMessage(message)
	if err != nil {
		return oops.
			In("kafka_consumer").
			With("topic", message.Topic).
			With("partition", message.Partition).
			With("offset", message.Offset).
			Time(time.Now()).
			Wrapf(err, "failed to decode Avro message")
	}

	// * Extract source information for table name
	source, ok := avroData["source"].(map[string]any)
	if !ok {
		return oops.
			In("kafka_consumer").
			With("topic", message.Topic).
			Time(time.Now()).
			New("source field not found or not a map")
	}

	tableName, ok := source["table"].(string)
	if !ok {
		return oops.
			In("kafka_consumer").
			With("topic", message.Topic).
			Time(time.Now()).
			New("table name not found in source")
	}

	// * Check if we have a handler for this table
	s.mu.RLock()
	handler, exists := s.handlers[tableName]
	s.mu.RUnlock()

	if !exists {
		s.l.Debug().
			Str("table", tableName).
			Msg("No handler registered for table, skipping")
		return nil
	}

	// * Convert Avro data to CDC event
	cdcEvent, err := s.convertAvroToCDCEvent(avroData)
	if err != nil {
		operation, _ := avroData["op"].(string)
		return oops.
			In("kafka_consumer").
			With("topic", message.Topic).
			With("table", tableName).
			With("operation", operation).
			Time(time.Now()).
			Wrapf(err, "failed to convert to CDC event")
	}

	// * Route to appropriate handler
	if err = handler.HandleEvent(cdcEvent); err != nil {
		return oops.
			In("kafka_consumer").
			With("topic", message.Topic).
			With("table", tableName).
			With("operation", cdcEvent.Operation).
			With("handler_type", fmt.Sprintf("%T", handler)).
			Time(time.Now()).
			Wrapf(err, "handler failed for table %s", tableName)
	}

	s.l.Debug().
		Str("table", tableName).
		Str("operation", cdcEvent.Operation).
		Msg("Successfully processed CDC event")

	return nil
}

// New helper function to extract a string field from a map.
func extractString(data map[string]any, key string) string {
	val, _ := data[key].(string)
	return val
}

// New helper to extract int64 from various possible Avro representations.
func extractInt64(field any) int64 {
	if field == nil {
		return 0
	}
	switch v := field.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case map[string]any:
		// Handle {"long": value} format
		if longVal, ok := v["long"]; ok {
			switch lv := longVal.(type) {
			case int64:
				return lv
			case float64:
				return int64(lv)
			}
		}
	}
	return 0
}

// New helper to extract and build the CDCSource from the source map.
func extractCDCSource(source map[string]any) cdctypes.CDCSource {
	var isSnapshot bool
	if snapshotField := source["snapshot"]; snapshotField != nil {
		switch v := snapshotField.(type) {
		case map[string]any:
			if snapshotVal, sOk := v["string"].(string); sOk {
				isSnapshot = snapshotVal != "false"
			}
		case string:
			isSnapshot = v != "false"
		}
	}

	return cdctypes.CDCSource{
		Database:  extractString(source, "db"),
		Schema:    extractString(source, "schema"),
		Table:     extractString(source, "table"),
		Connector: extractString(source, "connector"),
		Version:   extractString(source, "version"),
		Snapshot:  isSnapshot,
	}
}

// New helper to normalize the Debezium operation codes.
func normalizeOperation(op string) string {
	switch op {
	case "c":
		return "create"
	case "u":
		return "update"
	case "d":
		return "delete"
	case "r":
		return "read"
	default:
		return op
	}
}

// New helper to extract before/after state.
func extractDataState(avroData map[string]any, key string) map[string]any {
	var data map[string]any
	if dataField := avroData[key]; dataField != nil {
		if dataMap, ok := dataField.(map[string]any); ok {
			data = cdctypes.ExtractValueField(dataMap)
		}
	}
	for k, v := range data {
		data[k] = cdctypes.ConvertAvroOptionalField(v)
	}
	return data
}

// New helper to extract the transaction ID from the Avro data.
func extractTransactionID(avroData map[string]any) string {
	if txData, ok := avroData["transaction"].(map[string]any); ok {
		if txID, idOk := txData["id"].(string); idOk {
			return txID
		}
	}
	return ""
}

// convertAvroToCDCEvent transforms an Avro-decoded Debezium change event into our generic CDC event format.
// This handles the new Avro format where data is already decoded into maps.
//
// Parameters:
//   - avroData: The decoded Avro data containing the Debezium envelope.
//
// Returns:
//   - *services.CDCEvent: Normalized CDC event for handler processing.
//   - error: If event conversion fails.
func (s *KafkaConsumerService) convertAvroToCDCEvent(
	avroData map[string]any,
) (*cdctypes.CDCEvent, error) {
	op, ok := avroData["op"].(string)
	if !ok {
		return nil, eris.New("operation field not found or not a string")
	}

	sourceMap, ok := avroData["source"].(map[string]any)
	if !ok {
		return nil, eris.New("source field not found or not a map")
	}

	before := extractDataState(avroData, "before")
	after := extractDataState(avroData, "after")

	normalizedOp := normalizeOperation(op)
	source := extractCDCSource(sourceMap)
	transactionID := extractTransactionID(avroData)
	timestamp := extractInt64(avroData["ts_ms"])
	lsn := extractInt64(sourceMap["lsn"])

	return &cdctypes.CDCEvent{
		Operation: normalizedOp,
		Table:     source.Table,
		Schema:    source.Schema,
		Before:    before,
		After:     after,
		Metadata: cdctypes.CDCMetadata{
			Timestamp:     timestamp,
			TransactionID: transactionID,
			Source:        source,
			LSN:           lsn,
		},
	}, nil
}
