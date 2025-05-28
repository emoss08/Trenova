package rabbitmq

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type WorkflowParams struct {
	fx.In

	ConfigM *config.Manager
	Logger  *logger.Logger
}

type WorkflowPublisher struct {
	config       *config.RabbitMQConfig
	connection   *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	queueName    string
	l            *zerolog.Logger
}

func NewWorkflowPublisher(p WorkflowParams) *WorkflowPublisher {
	cfg := p.ConfigM.RabbitMQ()
	l := p.Logger.With().
		Str("service", "rabbitmq").Str("exchange", cfg.ExchangeName).
		Logger()

	l.Info().Str("computed_amqp_url", cfg.URL()).Msg("Attempting to connect to RabbitMQ with computed URL")

	conn, err := amqp.Dial(cfg.URL())
	if err != nil {
		l.Fatal().Err(err).Msg("failed to connect to rabbitmq")
		return nil
	}

	ch, err := conn.Channel()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to open a channel")
		return nil
	}

	// * Declare exchange
	err = ch.ExchangeDeclare(
		cfg.ExchangeName, // name
		"direct",         // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to declare exchange")
		return nil
	}

	// * Declare dead letter exchange
	err = ch.ExchangeDeclare(
		cfg.ExchangeName+".dlx", // name
		"direct",                // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to declare dead letter exchange")
		return nil
	}

	// * Declare queue
	_, err = ch.QueueDeclare(
		cfg.QueueName, // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		amqp.Table{
			"x-dead-letter-exchange": cfg.ExchangeName + ".dlx",
		}, // arguments
	)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to declare queue")
		return nil
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		cfg.QueueName,    // queue name
		cfg.QueueName,    // routing key
		cfg.ExchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to bind queue")
		return nil
	}

	l.Info().Msg("🚀 RabbitMQ publisher initialized")

	return &WorkflowPublisher{
		config:       cfg,
		connection:   conn,
		channel:      ch,
		exchangeName: cfg.ExchangeName,
		queueName:    cfg.QueueName,
		l:            &l,
	}
}

func (p *WorkflowPublisher) Publish(ctx context.Context, routingKey string, message any) error {
	msgBytes, err := sonic.Marshal(message)
	if err != nil {
		return eris.Wrap(err, "failed to marshal message")
	}

	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		p.exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         msgBytes,
		},
	)
	if err != nil {
		return eris.Wrap(err, "failed to publish message")
	}

	p.l.Info().
		Str("routingKey", routingKey).
		Str("exchange", p.exchangeName).
		Interface("message", message).
		Msg("message published")

	return nil
}

// Close closes the RabbitMQ connection and channel
func (p *WorkflowPublisher) Close() error {
	p.l.Info().Msg("closing RabbitMQ publisher connections")

	// Close the channel first
	if p.channel != nil {
		p.l.Debug().Msg("closing RabbitMQ channel")
		if err := p.channel.Close(); err != nil {
			p.l.Error().Err(err).Msg("failed to close RabbitMQ channel")
		} else {
			p.l.Debug().Msg("RabbitMQ channel closed successfully")
		}
	}

	// Then close the connection
	if p.connection != nil {
		p.l.Debug().Msg("closing RabbitMQ connection")
		if err := p.connection.Close(); err != nil {
			p.l.Error().Err(err).Msg("failed to close RabbitMQ connection")
			return eris.Wrap(err, "failed to close connection")
		}
		p.l.Debug().Msg("RabbitMQ connection closed successfully")
	}

	p.l.Info().Msg("RabbitMQ publisher connections closed successfully")
	return nil
}

// RegisterHooks registers lifecycle hooks for the publisher
func (p *WorkflowPublisher) RegisterHooks() fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			p.l.Info().Msg("🔴 Shutting down RabbitMQ publisher")

			// Use a short timeout to prevent hanging
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Channel to signal completion
			done := make(chan error, 1)

			// Close in a goroutine to respect timeout
			go func() {
				done <- p.Close()
			}()

			// Wait for close to complete or timeout
			select {
			case err := <-done:
				if err != nil {
					p.l.Error().Err(err).Msg("error during RabbitMQ publisher shutdown")
					return nil // Return nil anyway to allow shutdown to continue
				}
				p.l.Info().Msg("RabbitMQ publisher shut down successfully")
			case <-closeCtx.Done():
				p.l.Warn().Msg("RabbitMQ publisher shutdown timed out, forcing exit")
			}

			return nil
		},
	}
}
