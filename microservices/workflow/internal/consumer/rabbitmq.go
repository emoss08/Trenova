package consumer

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/microservices/workflow/internal/config"
	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rotisserie/eris"
)

type MessageHandler func(ctx context.Context, msg *model.Message) error

type RabbitMQConsumer struct {
	config         *config.RabbitMQConfig
	conn           *amqp.Connection
	ch             *amqp.Channel
	handlers       map[model.Type]MessageHandler
	reconnectDelay time.Duration
	mu             sync.RWMutex
	done           chan struct{}
	maxRetries     int
}

func NewRabbitMQConsumer(cfg *config.RabbitMQConfig) (*RabbitMQConsumer, error) {
	consumer := &RabbitMQConsumer{
		config:         cfg,
		handlers:       make(map[model.Type]MessageHandler),
		reconnectDelay: 5 * time.Second,
		done:           make(chan struct{}),
		maxRetries:     3, // Default max retries before message goes to DLX
	}

	return consumer, nil
}

func (c *RabbitMQConsumer) RegisterHandler(workflowType model.Type, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[workflowType] = handler
	log.Printf("Registered handler for workflow type: %s", workflowType)
}

// Start begins consuming messages from RabbitMQ
func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	var err error

	// Connect to RabbitMQ
	if err = c.connect(); err != nil {
		return eris.Wrap(err, "failed to connect to RabbitMQ")
	}

	// Set up the necessary exchanges, queues, and bindings
	if err = c.setupTopology(); err != nil {
		return eris.Wrap(err, "failed to set up RabbitMQ topology")
	}

	// Register workflow type handlers
	c.registerWorkflowBindings()

	// Start consuming messages
	go c.consumeMessages(ctx)

	return nil
}

func (c *RabbitMQConsumer) connect() error {
	var err error

	// * Try to connect to RabbitMQ with retries
	for range make([]struct{}, 5) {
		c.conn, err = amqp.Dial(c.config.URL())
		if err == nil {
			log.Printf("Connected to RabbitMQ")
			break
		}

		log.Printf("Failed to connect to RabbitMQ: %v", err)
		time.Sleep(c.reconnectDelay)
	}

	if err != nil {
		return eris.Wrap(err, "failed to connect to RabbitMQ")
	}

	// * Create a channel AFTER successful connection
	c.ch, err = c.conn.Channel()
	if err != nil {
		return eris.Wrap(err, "failed to open channel")
	}

	// Set QoS prefetch AFTER creating the channel
	err = c.ch.Qos(
		c.config.PrefetchCount,
		0,
		false,
	)
	if err != nil {
		return eris.Wrap(err, "failed to set QoS")
	}

	log.Printf("Connected to RabbitMQ and set QoS at %v", c.config.Host)
	return nil
}

func (c *RabbitMQConsumer) setupTopology() error {
	return c.setupTopologyWithRetry(false)
}

// setupTopologyWithRetry sets up the topology with an option to delete and recreate the queue
func (c *RabbitMQConsumer) setupTopologyWithRetry(deleteQueue bool) error {
	// Declare the main exchange
	err := c.ch.ExchangeDeclare(
		c.config.ExchangeName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return eris.Wrap(err, "failed to declare exchange")
	}

	// Declare the Dead Letter Exchange (DLX)
	dlxName := c.config.ExchangeName + ".dlx"
	err = c.ch.ExchangeDeclare(
		dlxName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return eris.Wrap(err, "failed to declare dead letter exchange")
	}

	// Declare the Dead Letter Queue
	dlqName := c.config.QueueName + ".dlq"
	dlq, err := c.ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return eris.Wrap(err, "failed to declare dead letter queue")
	}

	// Bind the DLQ to the DLX
	err = c.ch.QueueBind(
		dlq.Name,
		"#", // Catch all routing keys
		dlxName,
		false,
		nil,
	)
	if err != nil {
		return eris.Wrap(err, "failed to bind dead letter queue")
	}

	// If deleteQueue is true, try to delete the queue first
	if deleteQueue {
		log.Printf("Attempting to delete existing queue '%s'", c.config.QueueName)
		_, err = c.ch.QueueDelete(
			c.config.QueueName,
			false, // ifUnused
			false, // ifEmpty
			false, // noWait
		)
		if err != nil {
			log.Printf("Warning: Failed to delete queue '%s': %v", c.config.QueueName, err)
			// Continue anyway, as the queue might not exist
		}
	}

	// Declare main queue with dead letter configuration
	args := amqp.Table{
		"x-dead-letter-exchange": dlxName,
	}

	// Try to declare the queue with DLX configuration
	_, err = c.ch.QueueDeclare(
		c.config.QueueName,
		true,
		false,
		false,
		false,
		args,
	)
	
	// If queue exists with different arguments, try different approach
	if err != nil && isPreConditionFailedError(err) {
		log.Printf("Queue '%s' already exists with different parameters.", c.config.QueueName)
		
		// The channel might be closed after the precondition failure, so reconnect
		if err = c.reconnect(); err != nil {
			return eris.Wrap(err, "failed to reconnect")
		}
		
		// If this is our first attempt, try again with queue deletion enabled
		if !deleteQueue {
			log.Printf("Retrying with queue deletion...")
			return c.setupTopologyWithRetry(true)
		}
		
		// If we've already tried deleting, fall back to using the existing queue
		log.Printf("Using existing queue as-is (without dead letter exchange configuration)")
		
		// Try to bind the existing queue
		err = c.ch.QueueBind(
			c.config.QueueName,
			"#", // Catch all routing keys
			c.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return eris.Wrap(err, "failed to bind existing queue")
		}
		
	} else if err != nil {
		return eris.Wrap(err, "failed to declare queue")
	}

	log.Printf("Setup RabbitMQ topology with exchange %v, queue %v, and dead letter exchange %v with queue %v",
		c.config.ExchangeName, c.config.QueueName, dlxName, dlqName)
	return nil
}

// reconnect closes existing connections and reconnects to RabbitMQ
func (c *RabbitMQConsumer) reconnect() error {
	// Close existing channel and connection to ensure clean state
	if c.ch != nil {
		c.ch.Close() // Ignore errors as channel might already be closed
	}
	
	if c.conn != nil {
		c.conn.Close() // Ignore errors as connection might already be closed
	}
	
	// Reconnect to RabbitMQ
	return c.connect()
}

// isPreConditionFailedError checks if the error is a precondition failed error
func isPreConditionFailedError(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := err.Error()
	
	// Check if the error contains both "PRECONDITION_FAILED" and "x-dead-letter-exchange"
	// This covers the specific error we're seeing
	return strings.Contains(errMsg, "PRECONDITION_FAILED") && 
	       strings.Contains(errMsg, "x-dead-letter-exchange")
}

func (c *RabbitMQConsumer) registerWorkflowBindings() {
	// Get all available workflow types
	workflowTypes := c.getAvailableWorkflowTypes()

	for _, wType := range workflowTypes {
		// * Use the workflow type as the routing keys
		err := c.ch.QueueBind(
			c.config.QueueName,
			string(wType),
			c.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Warning: Failed to bind queue to exchange for workflow type %v: %v", wType, err)
			continue
		}
		log.Printf("Bound queue %v to exchange %v for workflow type %v", c.config.QueueName, c.config.ExchangeName, wType)
	}
}

// getAvailableWorkflowTypes returns all available workflow types
// This makes it easy to add new workflow types without modifying the binding code
func (c *RabbitMQConsumer) getAvailableWorkflowTypes() []model.Type {
	return []model.Type{
		model.TypeShipmentUpdated,
	}
}

// consumeMessages continuously consumes messages from the queue
func (c *RabbitMQConsumer) consumeMessages(ctx context.Context) {
	msgs, err := c.ch.Consume(
		c.config.QueueName, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		log.Printf("Failed to register consumer: %v", err)
		return
	}

	log.Printf("Started consuming messages from queue: %s", c.config.QueueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping consumer")
			return
		case <-c.done:
			log.Println("Consumer closed, stopping")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("Channel closed, attempting to reconnect...")
				time.Sleep(c.reconnectDelay)

				if err = c.connect(); err != nil {
					log.Printf("Failed to reconnect: %v", err)
					continue
				}

				if err = c.setupTopology(); err != nil {
					log.Printf("Failed to set up topology after reconnect: %v", err)
					continue
				}

				c.registerWorkflowBindings()

				msgs, err = c.ch.Consume(
					c.config.QueueName,
					"",
					false,
					false,
					false,
					false,
					nil,
				)
				if err != nil {
					log.Printf("Failed to register consumer after reconnect: %v", err)
				}
				continue
			}

			// Process the message
			go c.processMessage(ctx, msg)
		}
	}
}

// processMessage handles an individual message
func (c *RabbitMQConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in message processing: %v", r)
			c.handleMessageRetry(ctx, msg, nil)
		}
	}()

	log.Printf("Received message with routing key: %s", msg.RoutingKey)

	// Parse the message
	message, err := deserializeMessage(msg.Body)
	if err != nil {
		log.Printf("Failed to deserialize message: %v", err)
		// Nack the message without requeue as it's malformed
		if err = msg.Nack(false, false); err != nil {
			log.Printf("Failed to nack message: %v", err)
		}
		return
	}

	// Find the appropriate handler
	c.mu.RLock()
	handler, exists := c.handlers[message.Type]
	c.mu.RUnlock()

	if !exists {
		log.Printf("No handler registered for workflow type: %s", message.Type)
		// Ack the message since we can't process it
		if err = msg.Ack(false); err != nil {
			log.Printf("Failed to ack message: %v", err)
		}
		return
	}

	// Execute the handler
	err = handler(ctx, message)
	if err != nil {
		log.Printf("Error handling message: %v", err)
		c.handleMessageRetry(ctx, msg, err)
		return
	}

	// Acknowledge the message
	if err = msg.Ack(false); err != nil {
		log.Printf("Failed to ack message: %v", err)
	}
}

// handleMessageRetry manages the retry logic for failed messages
func (c *RabbitMQConsumer) handleMessageRetry(ctx context.Context, msg amqp.Delivery, processingErr error) {
	retryCount := c.getRetryCount(msg)

	// Increment retry count
	retryCount++

	// Check if we've exceeded max retries
	if retryCount > c.maxRetries {
		c.sendToDeadLetterQueue(msg)
		return
	}

	// Republish with updated retry count
	c.republishWithRetryCount(ctx, msg, retryCount)
}

// getRetryCount extracts the retry count from message headers
func (c *RabbitMQConsumer) getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}

	count, exists := msg.Headers["x-retry-count"]
	if !exists {
		return 0
	}

	countInt, ok := count.(int)
	if !ok {
		return 0
	}

	return countInt
}

// sendToDeadLetterQueue rejects a message to send it to the DLQ
func (c *RabbitMQConsumer) sendToDeadLetterQueue(msg amqp.Delivery) {
	log.Printf("Message exceeded maximum retry count (%d). Sending to dead letter queue.", c.maxRetries)
	if err := msg.Nack(false, false); err != nil {
		log.Printf("Failed to nack message to dead letter queue: %v", err)
	}
}

// republishWithRetryCount republishes a message with updated retry count
func (c *RabbitMQConsumer) republishWithRetryCount(ctx context.Context, msg amqp.Delivery, retryCount int) {
	headers := amqp.Table{}
	if msg.Headers != nil {
		headers = msg.Headers
	}
	headers["x-retry-count"] = retryCount

	publishErr := c.ch.PublishWithContext(
		ctx,
		c.config.ExchangeName,
		msg.RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			Headers:         headers,
			ContentType:     msg.ContentType,
			ContentEncoding: msg.ContentEncoding,
			Body:            msg.Body,
			DeliveryMode:    msg.DeliveryMode,
			Priority:        msg.Priority,
		},
	)

	if publishErr != nil {
		log.Printf("Failed to republish message with retry count: %v", publishErr)
		// Fallback to regular nack if republish fails
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("Failed to nack message: %v", nackErr)
		}
	} else {
		log.Printf("Republished message with retry count: %d", retryCount)
		// Ack the original message since we've republished it
		if ackErr := msg.Ack(false); ackErr != nil {
			log.Printf("Failed to ack original message after republishing: %v", ackErr)
		}
	}
}

// Stop gracefully shuts down the consumer
func (c *RabbitMQConsumer) Stop() error {
	close(c.done)

	if c.ch != nil {
		if err := c.ch.Close(); err != nil {
			return eris.Wrap(err, "failed to close channel")
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return eris.Wrap(err, "failed to close connection")
		}
	}

	log.Println("RabbitMQ consumer stopped")
	return nil
}

// Helper functions
func deserializeMessage(data []byte) (*model.Message, error) {
	var message model.Message

	if err := sonic.Unmarshal(data, &message); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal message")
	}

	return &message, nil
}
