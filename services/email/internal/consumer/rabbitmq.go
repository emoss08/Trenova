/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package consumer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/microservices/email/internal/config"
	"github.com/emoss08/trenova/microservices/email/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rotisserie/eris"
)

type MessageHandler func(ctx context.Context, msg *model.Message) error

// RabbitMQConsumer is a consumer for RabbitMQ messages
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

// NewRabbitMQConsumer creates a new RabbitMQ consumer
func NewRabbitMQConsumer(cfg *config.AppConfig) (*RabbitMQConsumer, error) {
	consumer := &RabbitMQConsumer{
		config:         &cfg.RabbitMQ,
		handlers:       make(map[model.Type]MessageHandler),
		reconnectDelay: 5 * time.Second,
		done:           make(chan struct{}),
		maxRetries:     1,
	}

	return consumer, nil
}

// RegisterHandler registers a handler for a message type
func (c *RabbitMQConsumer) RegisterHandler(messageType model.Type, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[messageType] = handler
	log.Printf("Registered handler for message type: %s", messageType)
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

	// Register message type bindings
	c.registerMessageBindings()

	// Start consuming messages
	go c.consumeMessages(ctx)

	return nil
}

func (c *RabbitMQConsumer) connect() error {
	var err error

	// Try to connect to RabbitMQ with retries
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

	// Create a channel AFTER successful connection
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

	// Declare main queue with dead letter configuration
	args := amqp.Table{
		"x-dead-letter-exchange": dlxName,
	}

	_, err = c.ch.QueueDeclare(
		c.config.QueueName,
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		return eris.Wrap(err, "failed to declare main queue")
	}

	log.Printf(
		"Setup RabbitMQ topology with exchange %v, queue %v, and dead letter exchange %v with queue %v",
		c.config.ExchangeName,
		c.config.QueueName,
		dlxName,
		dlqName,
	)

	return nil
}

func (c *RabbitMQConsumer) registerMessageBindings() {
	// Get all available message types
	messageTypes := c.getAvailableMessageTypes()

	for _, mType := range messageTypes {
		// Use the message type as the routing keys
		err := c.ch.QueueBind(
			c.config.QueueName,
			string(mType),
			c.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			log.Printf(
				"Warning: Failed to bind queue to exchange for message type %v: %v",
				mType,
				err,
			)
			continue
		}
		log.Printf(
			"Bound queue %v to exchange %v for message type %v",
			c.config.QueueName,
			c.config.ExchangeName,
			mType,
		)
	}
}

// getAvailableMessageTypes returns all available message types
func (c *RabbitMQConsumer) getAvailableMessageTypes() []model.Type {
	return []model.Type{
		model.TypeEmailSend,
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

				c.registerMessageBindings()

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
			c.handleMessageRetry(ctx, msg)
		}
	}()

	log.Printf("Received message with routing key: %s", msg.RoutingKey)

	// Log message body for debugging
	log.Printf("Message body: %s", string(msg.Body))

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

	// Log deserialized message details for debugging
	log.Printf(
		"Deserialized message: ID=%s, Type=%s, TenantID=%s",
		message.ID,
		message.Type,
		message.TenantID,
	)

	// Find the appropriate handler
	c.mu.RLock()
	handler, exists := c.handlers[message.Type]
	c.mu.RUnlock()

	if !exists {
		log.Printf("No handler registered for message type: %s", message.Type)
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
		c.handleMessageRetry(ctx, msg)
		return
	}

	// Acknowledge the message
	if err = msg.Ack(false); err != nil {
		log.Printf("Failed to ack message: %v", err)
	}
}

// handleMessageRetry manages the retry logic for failed messages
func (c *RabbitMQConsumer) handleMessageRetry(ctx context.Context, msg amqp.Delivery) {
	retryCount := c.getRetryCount(msg)

	// Increment retry count
	retryCount++

	// Check if we've exceeded max retries
	if retryCount >= c.maxRetries {
		log.Printf(
			"Message exceeded maximum retry count (%d). Sending to dead letter queue.",
			c.maxRetries,
		)
		if err := msg.Nack(false, false); err != nil {
			log.Printf("Failed to nack message to dead letter queue: %v", err)
		}
		return
	}

	// Republish with updated retry count
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
