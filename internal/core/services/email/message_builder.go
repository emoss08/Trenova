package email

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/services/email/providers"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// MessageBuilder builds email messages
type MessageBuilder interface {
	BuildMessage(
		ctx context.Context,
		profile *email.Profile,
		queue *email.Queue,
	) (*providers.Message, error)
}

type messageBuilder struct {
	l                 *zerolog.Logger
	attachmentHandler AttachmentHandler
}

type MessageBuilderParams struct {
	fx.In

	Logger            *logger.Logger
	AttachmentHandler AttachmentHandler
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder(p MessageBuilderParams) MessageBuilder {
	log := p.Logger.With().
		Str("component", "message_builder").
		Logger()

	return &messageBuilder{
		l:                 &log,
		attachmentHandler: p.AttachmentHandler,
	}
}

// BuildMessage builds an email message from profile and queue data
func (b *messageBuilder) BuildMessage(
	ctx context.Context,
	profile *email.Profile,
	queue *email.Queue,
) (*providers.Message, error) {
	log := b.l.With().
		Str("operation", "build_message").
		Str("queue_id", queue.ID.String()).
		Logger()
	msg := &providers.Message{
		From: providers.EmailAddress{
			Email: profile.FromAddress,
			Name:  profile.FromName,
		},
		Subject:  queue.Subject,
		HTMLBody: queue.HTMLBody,
		TextBody: queue.TextBody,
		Priority: queue.Priority,
		Headers:  make(map[string]string),
	}

	// Set reply-to if configured
	if profile.ReplyTo != "" {
		msg.ReplyTo = &providers.EmailAddress{
			Email: profile.ReplyTo,
		}
	}

	// Convert recipients
	msg.To = b.convertAddresses(queue.ToAddresses)
	msg.CC = b.convertAddresses(queue.CCAddresses)
	msg.BCC = b.convertAddresses(queue.BCCAddresses)

	// Add metadata headers
	if queue.Metadata != nil {
		if orgID, ok := queue.Metadata["organization_id"].(string); ok {
			msg.Headers["X-Organization-ID"] = orgID
		}
		if queueID, ok := queue.Metadata["queue_id"].(string); ok {
			msg.Headers["X-Queue-ID"] = queueID
		}
	}

	// Convert attachments
	if len(queue.Attachments) > 0 {
		log.Debug().Int("attachment_count", len(queue.Attachments)).Msg("processing attachments")

		attachments, err := b.convertAttachments(ctx, queue.Attachments, queue.OrganizationID)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert attachments")
			return nil, oops.In("message_builder").
				Tags("operation", "convert_attachments").
				Tags("queue_id", queue.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to convert attachments")
		}
		msg.Attachments = attachments
	}

	return msg, nil
}

// convertAddresses converts string addresses to EmailAddress structs
func (b *messageBuilder) convertAddresses(addresses []string) []providers.EmailAddress {
	result := make([]providers.EmailAddress, len(addresses))
	for i, addr := range addresses {
		result[i] = providers.EmailAddress{
			Email: addr,
		}
	}
	return result
}

// convertAttachments converts email attachments to provider attachments
func (b *messageBuilder) convertAttachments(
	ctx context.Context,
	attachments []email.AttachmentMeta,
	orgID pulid.ID,
) ([]providers.Attachment, error) {
	log := b.l.With().
		Str("operation", "convert_attachments").
		Int("attachment_count", len(attachments)).
		Logger()

	if len(attachments) == 0 {
		return nil, nil
	}

	// Validate attachments first
	if err := b.attachmentHandler.ValidateAttachments(attachments); err != nil {
		log.Error().Err(err).Msg("attachment validation failed")
		return nil, oops.In("message_builder").
			Tags("operation", "validate_attachments").
			Time(time.Now()).
			Wrapf(err, "attachment validation failed")
	}

	result := make([]providers.Attachment, 0, len(attachments))

	for i, attachment := range attachments {
		log := log.With().
			Int("attachment_index", i).
			Str("filename", attachment.FileName).
			Logger()

		// Get attachment data from storage
		data, err := b.attachmentHandler.GetAttachmentData(ctx, &attachment, orgID)
		if err != nil {
			log.Error().Err(err).Msg("failed to get attachment data")
			return nil, oops.In("message_builder").
				Tags("operation", "get_attachment_data").
				Tags("filename", attachment.FileName).
				Time(time.Now()).
				Wrapf(err, "failed to get attachment data")
		}

		// Convert to provider attachment
		providerAttachment := providers.Attachment{
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			Data:        data,
			ContentID:   attachment.ContentID,
		}

		result = append(result, providerAttachment)

		log.Debug().
			Int("data_size", len(data)).
			Msg("attachment converted successfully")
	}

	log.Info().
		Int("converted_count", len(result)).
		Msg("all attachments converted successfully")

	return result, nil
}
