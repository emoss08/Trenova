package cdc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/pkg/cdctypes"
	"github.com/emoss08/trenova/pkg/utils/cdcutils"
	"github.com/emoss08/trenova/pkg/utils/maputils"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
	"github.com/sourcegraph/conc"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type KafkaConsumerParams struct {
	fx.In

	Logger          *zap.Logger
	Config          *config.Config
	MetricsRegistry *observability.MetricsRegistry `optional:"true"`
}

type KafkaConsumer struct {
	l            *zap.Logger
	config       *config.CDCConfig
	reader       *kafka.Reader
	ctx          context.Context
	cancel       context.CancelFunc
	wg           *conc.WaitGroup
	running      atomic.Bool
	handlers     sync.Map
	schemaClient *srclient.SchemaRegistryClient
	schemaCache  *lru.Cache[int, *goavro.Codec]
	metrics      *observability.MetricsRegistry
	messageQueue chan *kafka.Message
	workerPool   *pool.Pool
}

func NewKafkaConsumer(p KafkaConsumerParams) services.CDCService {
	cfg := p.Config.CDC
	schemaClient := srclient.CreateSchemaRegistryClient(cfg.SchemaRegistryURL)

	schemaCache, err := lru.New[int, *goavro.Codec](cfg.SchemaCache.MaxSize)
	if err != nil {
		p.Logger.Fatal("Failed to create schema cache", zap.Error(err))
	}

	consumer := &KafkaConsumer{
		l:            p.Logger.With(zap.String("service", "kafka-consumer")),
		config:       cfg,
		wg:           conc.NewWaitGroup(),
		schemaClient: schemaClient,
		schemaCache:  schemaCache,
		metrics:      p.MetricsRegistry,
	}

	if cfg.Processing.EnableParallelProcessing {
		consumer.messageQueue = make(chan *kafka.Message, cfg.Processing.MessageChannelSize)
	}

	return consumer
}

func (s *KafkaConsumer) RegisterHandler(table string, handler services.CDCEventHandler) {
	s.handlers.Store(table, handler)
	s.l.Info(
		"Registered CDC handler for table",
		zap.String("table", table),
		zap.String("handler_type", fmt.Sprintf("%T", handler)),
	)
}

func (s *KafkaConsumer) Start() error {
	if s.running.Load() {
		return cdctypes.ErrConsumerAlreadyRunning
	}

	if !s.config.Enabled {
		s.l.Info("Kafka consumer disabled via configuration")
		return nil
	}

	s.l.Info(
		"Starting Kafka consumer",
		zap.Strings("brokers", s.config.Brokers),
		zap.String("topic_pattern", s.config.TopicPattern),
		zap.String("group_id", s.config.ConsumerGroup),
	)

	// ! Create a background context for long-running operations instead of using the startup context
	// ! The startup context has a timeout and will cancel after application starts
	s.ctx, s.cancel = context.WithCancel(context.Background())

	startOffset := kafka.LastOffset
	if strings.EqualFold(s.config.StartOffset, "earliest") {
		startOffset = kafka.FirstOffset
	}

	topics, err := s.getMatchingTopics()
	if err != nil {
		s.l.Warn("Failed to get matching topics, will retry later", zap.Error(err))
		topics = s.getDefaultTopics()
	}

	if len(topics) == 0 {
		s.l.Warn("No matching topics found, will retry later")
		topics = s.getDefaultTopics()
	}

	s.l.Debug("Subscribing to topics", zap.Strings("topics", topics))

	if len(topics) > 0 {
		s.l.Info("Starting consumer for topics", zap.Strings("topics", topics))

		s.reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers:                s.config.Brokers,
			GroupID:                s.config.ConsumerGroup,
			GroupTopics:            topics, // Subscribe to all matching topics
			StartOffset:            startOffset,
			MinBytes:               1,
			MaxBytes:               10e6,
			CommitInterval:         1 * time.Second,
			MaxWait:                1 * time.Second,
			ReadBatchTimeout:       10 * time.Second,
			QueueCapacity:          1000,
			HeartbeatInterval:      3 * time.Second,
			SessionTimeout:         30 * time.Second,
			RebalanceTimeout:       30 * time.Second,
			PartitionWatchInterval: 30 * time.Second,
			MaxAttempts:            3,
			Dialer: &kafka.Dialer{
				Timeout:   30 * time.Second,
				DualStack: true,
				KeepAlive: 30 * time.Second,
			},
			ErrorLogger: kafka.LoggerFunc(s.logKafkaError),
		})
	}

	s.running.Store(true)

	if s.config.Processing.EnableParallelProcessing {
		s.startWorkerPool()
		s.l.Info(
			"Started worker pool for parallel processing",
			zap.Int("worker_count", s.config.Processing.WorkerCount),
			zap.Int("queue_size", s.config.Processing.MessageChannelSize),
		)
	}

	s.wg.Go(func() {
		s.consumeMessages()
	})

	s.l.Info(
		"Kafka consumer started",
		zap.Bool("parallel_processing", s.config.Processing.EnableParallelProcessing),
		zap.Int("workers", s.config.Processing.WorkerCount),
	)
	return nil
}

func (s *KafkaConsumer) Stop() error {
	if !s.running.Load() {
		return nil
	}

	s.l.Info("Stopping Kafka consumer")

	if s.cancel != nil {
		s.cancel()
	}

	if s.config.Processing.EnableParallelProcessing && s.messageQueue != nil {
		close(s.messageQueue)
		s.l.Debug("Closed message queue")
	}

	if s.workerPool != nil {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			s.config.Processing.ShutdownTimeout,
		)
		defer cancel()

		done := make(chan struct{})
		go func() {
			s.workerPool.Wait()
			close(done)
		}()

		select {
		case <-done:
			s.l.Info("Worker pool shut down gracefully")
		case <-shutdownCtx.Done():
			s.l.Warn(
				"Worker pool shutdown timed out",
				zap.Duration("timeout", s.config.Processing.ShutdownTimeout),
			)
		}
	}

	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			s.l.Error("Error closing Kafka reader", zap.Error(err))
		}
	}

	s.wg.Wait()
	s.running.Store(false)

	s.l.Info("Kafka consumer stopped")
	return nil
}

func (s *KafkaConsumer) IsRunning() bool {
	return s.running.Load()
}

func (s *KafkaConsumer) startWorkerPool() {
	s.workerPool = pool.New().WithMaxGoroutines(s.config.Processing.WorkerCount)

	if s.metrics != nil {
		s.metrics.UpdateCDCProcessingWorkers(s.config.Processing.WorkerCount)
	}

	s.wg.Go(func() {
		s.processMessageQueue()
	})
}

func (s *KafkaConsumer) processMessageQueue() {
	for {
		select {
		case <-s.ctx.Done():
			s.l.Debug("Message queue processor stopping")
			return
		case message, ok := <-s.messageQueue:
			if !ok {
				s.l.Debug("Message queue closed")
				return
			}

			msg := message
			s.workerPool.Go(func() {
				s.processAndLogMessage(msg)
			})
		}
	}
}

func (s *KafkaConsumer) retryWithBackoff(
	ctx context.Context,
	operation func() error,
	operationName string,
) error {
	backoff := s.config.Retry.InitialBackoff
	maxAttempts := s.config.Retry.MaxAttempts

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return fmt.Errorf("%s cancelled: %w", operationName, ctx.Err())
		}

		if attempt == maxAttempts {
			return fmt.Errorf("%s failed after %d attempts: %w", operationName, maxAttempts, err)
		}

		s.l.Warn(
			"Operation failed, retrying",
			zap.String("operation", operationName),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = time.Duration(float64(backoff) * s.config.Retry.BackoffFactor)
			if backoff > s.config.Retry.MaxBackoff {
				backoff = s.config.Retry.MaxBackoff
			}
		}
	}

	return nil
}

func (s *KafkaConsumer) logKafkaError(msg string, args ...any) {
	if strings.Contains(msg, "Rebalance In Progress") || strings.Contains(msg, "[27]") {
		s.l.Debug("Kafka rebalancing", zap.String("message", msg), zap.Any("args", args))
		if s.metrics != nil {
			s.metrics.RecordCDCRebalance()
		}
		return
	}
	if strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "timeout") {
		s.l.Debug(
			"Kafka timeout (normal when idle)",
			zap.String("message", msg),
			zap.Any("args", args),
		)
		return
	}
	s.l.Error("Kafka error", zap.String("message", msg), zap.Any("args", args))
}

func (s *KafkaConsumer) getMatchingTopics() ([]string, error) {
	conn, err := kafka.Dial("tcp", s.config.Brokers[0])
	if err != nil {
		s.l.Error("Failed to connect to Kafka", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		s.l.Error("Failed to read partitions", zap.Error(err))
		return nil, fmt.Errorf("failed to read partitions: %w", err)
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

func (s *KafkaConsumer) convertPatternToRegex(pattern string) *regexp.Regexp {
	regexPattern := strings.ReplaceAll(pattern, "*", ".*")
	regexPattern = "^" + regexPattern + "$"

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		s.l.Warn("Invalid pattern, using match-all", zap.Error(err), zap.String("pattern", pattern))
		return regexp.MustCompile(".*")
	}

	return regex
}

func (s *KafkaConsumer) getDefaultTopics() []string {
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

func (s *KafkaConsumer) handleReadError(
	err error,
	topic string,
	backoff *time.Duration,
) bool {
	if errors.Is(err, context.Canceled) {
		s.l.Info("Consumer context cancelled during read", zap.String("topic", topic))
		return false
	}

	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		s.l.Debug("Read timeout - no new messages (normal)", zap.String("topic", topic))
		*backoff = 0 // Reset backoff on normal timeout
		return true
	}

	const maxBackoff = 30 * time.Second
	if *backoff == 0 {
		*backoff = 1 * time.Second
	} else {
		*backoff *= 2
		if *backoff > maxBackoff {
			*backoff = maxBackoff
		}
	}
	s.l.Warn(
		"Kafka read error, backing off",
		zap.Error(err),
		zap.String("topic", topic),
		zap.Duration("backoff", *backoff),
	)
	time.Sleep(*backoff)

	return true
}

func (s *KafkaConsumer) processAndLogMessage(message *kafka.Message) {
	processCtx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	err := s.processMessage(processCtx, message)
	duration := time.Since(start).Seconds()

	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordCDCMessage("unknown", "unknown", "error", duration)
		}
	}
}

func (s *KafkaConsumer) decodeAvroMessage(message *kafka.Message) (map[string]any, error) {
	// ! Kafka Connect Avro messages have the format: [magic_byte][schema_id][avro_data]
	// ! The first byte is 0x0 (magic byte), followed by 4 bytes for schema ID
	if len(message.Value) < 5 {
		return nil, ErrMessageTooShort
	}

	schemaID := int(
		message.Value[1],
	)<<24 | int(
		message.Value[2],
	)<<16 | int(
		message.Value[3],
	)<<8 | int(
		message.Value[4],
	)

	codec, ok := s.schemaCache.Get(schemaID)
	if !ok { //nolint:nestif // this is fine
		if s.metrics != nil {
			s.metrics.RecordCDCSchemaCache("miss")
		}

		schema, err := s.schemaClient.GetSchema(schemaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema by ID %d: %w", schemaID, err)
		}

		codec, err = goavro.NewCodec(schema.Schema())
		if err != nil {
			return nil, fmt.Errorf("failed to create Avro codec for schema %d: %w", schemaID, err)
		}

		evicted := s.schemaCache.Add(schemaID, codec)
		if evicted && s.metrics != nil {
			s.metrics.RecordCDCSchemaCache("eviction")
		}
	} else if s.metrics != nil {
		s.metrics.RecordCDCSchemaCache("hit")
	}

	native, _, err := codec.NativeFromBinary(message.Value[5:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode Avro message: %w", err)
	}

	if result, rOk := native.(map[string]any); rOk {
		return result, nil
	}

	return nil, ErrDecodeAvroMessage
}

func (s *KafkaConsumer) processMessage( //nolint:funlen // this is fine
	ctx context.Context,
	message *kafka.Message,
) error {
	log := s.l.With(zap.String("operation", "processMessage"))
	start := time.Now()

	avroData, err := s.decodeAvroMessage(message)
	if err != nil {
		if errors.Is(err, ErrMessageTooShort) {
			log.Warn(
				"Message too short to be an Avro message",
				zap.String("topic", message.Topic),
				zap.Int("partition", message.Partition),
				zap.Int64("offset", message.Offset),
				zap.Int("message_size", len(message.Value)),
			)
			return nil
		}

		log.Error(
			"Failed to decode Avro message",
			zap.Error(err),
			zap.String("topic", message.Topic),
			zap.Int("partition", message.Partition),
			zap.Int64("offset", message.Offset),
			zap.Int("message_size", len(message.Value)),
		)
		if s.metrics != nil {
			s.metrics.RecordCDCHandlerError("unknown", "decode_error")
		}
		return fmt.Errorf("failed to decode Avro message: %w", err)
	}

	source, ok := avroData["source"].(map[string]any)
	if !ok {
		sourceErr := errors.New("source field not found or not a map")
		log.Error(
			"Invalid CDC message structure",
			zap.Error(sourceErr),
			zap.String("topic", message.Topic),
			zap.Int("partition", message.Partition),
			zap.Int64("offset", message.Offset),
		)
		if s.metrics != nil {
			s.metrics.RecordCDCHandlerError("unknown", "invalid_structure")
		}
		return sourceErr
	}

	tableName, ok := source["table"].(string)
	if !ok {
		tableErr := errors.New("table name not found in source")
		log.Error(
			"Table name missing from CDC message",
			zap.Error(tableErr),
			zap.String("topic", message.Topic),
			zap.Int("partition", message.Partition),
			zap.Int64("offset", message.Offset),
		)
		if s.metrics != nil {
			s.metrics.RecordCDCHandlerError("unknown", "missing_table")
		}
		return tableErr
	}

	handler, exists := s.handlers.Load(tableName)
	if !exists {
		return fmt.Errorf("handler does not exist for table %s", tableName)
	}

	cdcEvent, err := s.convertAvroToCDCEvent(avroData)
	if err != nil {
		log.Error(
			"Failed to convert to CDC event",
			zap.Error(err),
			zap.String("table", tableName),
		)
		if s.metrics != nil {
			s.metrics.RecordCDCHandlerError(tableName, "conversion_error")
		}
		return fmt.Errorf("failed to convert to CDC event: %w", err)
	}

	handlerErr := s.retryWithBackoff(ctx, func() error {
		if err = handler.(services.CDCEventHandler).HandleEvent(ctx, cdcEvent); err != nil { //nolint:errcheck // we're returning the error
			return err
		}

		return nil
	}, fmt.Sprintf("handler for table %s", tableName))

	if handlerErr != nil {
		log.Error(
			"Handler failed after retries",
			zap.Error(handlerErr),
			zap.String("table", tableName),
			zap.String("operation", cdcEvent.Operation),
			zap.Int("max_attempts", s.config.Retry.MaxAttempts),
		)
		if s.metrics != nil {
			duration := time.Since(start).Seconds()
			s.metrics.RecordCDCMessage(tableName, cdcEvent.Operation, "error", duration)
			s.metrics.RecordCDCHandlerError(tableName, "handler_failed")
		}

		return fmt.Errorf("handler failed for table %s after retries: %w", tableName, handlerErr)
	}

	duration := time.Since(start).Seconds()
	if s.metrics != nil {
		s.metrics.RecordCDCMessage(tableName, cdcEvent.Operation, "success", duration)
	}

	log.Debug(
		"Successfully processed CDC event",
		zap.String("table", tableName),
		zap.String("operation", cdcEvent.Operation),
		zap.Float64("duration_seconds", duration),
	)

	return nil
}

func (s *KafkaConsumer) convertAvroToCDCEvent(
	avroData map[string]any,
) (*cdctypes.CDCEvent, error) {
	op, ok := avroData["op"].(string)
	if !ok {
		return nil, ErrOperationFieldNotFound
	}

	sourceMap, ok := avroData["source"].(map[string]any)
	if !ok {
		return nil, ErrSourceFieldNotFound
	}

	before := cdcutils.ExtractDataState(avroData, "before")
	after := cdcutils.ExtractDataState(avroData, "after")

	normalizedOp := cdcutils.NormalizeOperation(op)
	source := cdcutils.ExtractCDCSource(sourceMap)
	transactionID := cdcutils.ExtractTransactionID(avroData)
	timestamp := maputils.ExtractInt64Field(avroData["ts_ms"])
	lsn := maputils.ExtractInt64Field(sourceMap["lsn"])

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

func (s *KafkaConsumer) consumeMessages() {
	s.l.Info(
		"Started message consumer",
		zap.Bool("parallel_processing", s.config.Processing.EnableParallelProcessing),
	)

	var backoff time.Duration

	for {
		select {
		case <-s.ctx.Done():
			s.l.Info("Message consumer stopped")
			return

		default:
			message, err := s.reader.ReadMessage(s.ctx)
			if err != nil {
				if !s.handleReadError(err, "multi-topic", &backoff) {
					return
				}
				continue
			}

			backoff = 0

			if s.config.Processing.EnableParallelProcessing {
				select {
				case s.messageQueue <- &message:
				case <-s.ctx.Done():
					return
				default:
					s.l.Warn(
						"Message queue full, processing synchronously",
						zap.Int("queue_size", s.config.Processing.MessageChannelSize),
					)
					s.processAndLogMessage(&message)
				}
			} else {
				s.processAndLogMessage(&message)
			}
		}
	}
}
