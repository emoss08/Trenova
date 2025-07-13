package email

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// Sender handles the actual email sending logic
type Sender interface {
	Send(ctx context.Context, profile *email.Profile, queue *email.Queue) (string, error)
	TestConnection(ctx context.Context, profile *email.Profile) error
}

type sender struct {
	l              *zerolog.Logger
	clientFactory  ClientFactory
	messageBuilder MessageBuilder
}

type SenderParams struct {
	fx.In

	Logger         *logger.Logger
	ClientFactory  ClientFactory
	MessageBuilder MessageBuilder
}

// NewSender creates a new email sender
func NewSender(p SenderParams) Sender {
	log := p.Logger.With().
		Str("component", "email_sender").
		Logger()

	return &sender{
		l:              &log,
		clientFactory:  p.ClientFactory,
		messageBuilder: p.MessageBuilder,
	}
}

// Send sends an email using the specified profile
func (s *sender) Send(
	ctx context.Context,
	profile *email.Profile,
	queue *email.Queue,
) (string, error) {
	log := s.l.With().
		Str("operation", "send_email").
		Str("profile_id", profile.ID.String()).
		Str("queue_id", queue.ID.String()).
		Logger()

	// Get the provider for this profile
	provider, err := s.clientFactory.GetProvider(profile)
	if err != nil {
		log.Error().Err(err).Msg("failed to get provider")
		return "", oops.In("sender").
			Tags("operation", "get_provider").
			Tags("provider_type", string(profile.ProviderType)).
			Time(time.Now()).
			Wrapf(err, "failed to get provider")
	}

	// Build provider configuration
	config, err := s.clientFactory.BuildConfig(profile)
	if err != nil {
		log.Error().Err(err).Msg("failed to build config")
		return "", oops.In("sender").
			Tags("operation", "build_config").
			Tags("profile_id", profile.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to build config")
	}

	// Build message
	msg, err := s.messageBuilder.BuildMessage(ctx, profile, queue)
	if err != nil {
		log.Error().Err(err).Msg("failed to build message")
		return "", oops.In("sender").
			Tags("operation", "build_message").
			Tags("queue_id", queue.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to build message")
	}

	// Log sending attempt
	log.Debug().
		Strs("to", queue.ToAddresses).
		Str("subject", queue.Subject).
		Str("provider", string(profile.ProviderType)).
		Msg("sending email")

	// Send the email using the provider
	messageID, err := provider.Send(ctx, config, msg)
	if err != nil {
		log.Error().Err(err).Msg("failed to send email")
		return "", oops.In("sender").
			Tags("operation", "send").
			Tags("provider", string(profile.ProviderType)).
			Time(time.Now()).
			Wrapf(err, "failed to send email")
	}

	log.Info().
		Str("message_id", messageID).
		Msg("email sent successfully")

	return messageID, nil
}

// TestConnection tests the email profile configuration
func (s *sender) TestConnection(ctx context.Context, profile *email.Profile) error {
	log := s.l.With().
		Str("operation", "test_connection").
		Str("profile_id", profile.ID.String()).
		Str("provider", string(profile.ProviderType)).
		Logger()

	// Get the provider for this profile
	provider, err := s.clientFactory.GetProvider(profile)
	if err != nil {
		log.Error().Err(err).Msg("failed to get provider")
		return oops.In("sender").
			Tags("operation", "get_provider").
			Tags("provider_type", string(profile.ProviderType)).
			Time(time.Now()).
			Wrapf(err, "failed to get provider")
	}

	// Build provider configuration
	config, err := s.clientFactory.BuildConfig(profile)
	if err != nil {
		log.Error().Err(err).Msg("failed to build config")
		return oops.In("sender").
			Tags("operation", "build_config").
			Tags("profile_id", profile.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to build config")
	}

	// Test the connection using the provider
	if err := provider.TestConnection(ctx, config); err != nil {
		log.Error().Err(err).Msg("connection test failed")
		return oops.In("sender").
			Tags("operation", "test_connection").
			Tags("provider", string(profile.ProviderType)).
			Time(time.Now()).
			Wrapf(err, "connection test failed")
	}

	log.Info().Msg("connection test successful")
	return nil
}
