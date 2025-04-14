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

	conn, err := amqp.Dial(p.ConfigM.RabbitMQ().URL())
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
	if err := p.channel.Close(); err != nil {
		return eris.Wrap(err, "failed to close channel")
	}
	if err := p.connection.Close(); err != nil {
		return eris.Wrap(err, "failed to close connection")
	}
	return nil
}

// RegisterHooks registers lifecycle hooks for the publisher
func (p *WorkflowPublisher) RegisterHooks() fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			p.l.Info().Msg("🔴 Closing RabbitMQ publisher")
			return p.Close()
		},
	}
}
