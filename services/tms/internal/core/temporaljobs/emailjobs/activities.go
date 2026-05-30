package emailjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	EmailService services.EmailService
	Logger       *zap.Logger
}

type Activities struct {
	emailService services.EmailService
	logger       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		emailService: p.EmailService,
		logger:       p.Logger.Named("email-activities"),
	}
}

func (a *Activities) SendEmailActivity(
	ctx context.Context,
	payload *SendEmailPayload,
) (*SendEmailResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Starting email send activity",
		"messageId", payload.MessageID.String(),
		"organizationId", payload.OrganizationID.String(),
		"businessUnitId", payload.BusinessUnitID.String(),
	)

	activity.RecordHeartbeat(ctx, "sending email")

	msg, err := a.emailService.SendPersisted(ctx, &services.SendPersistedEmailRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
		MessageID: payload.MessageID,
		HTML:      payload.HTML,
		Text:      payload.Text,
	})
	if err != nil {
		logger.Error("Failed to send email", "messageId", payload.MessageID.String(), "error", err)
		result := &SendEmailResult{
			MessageID: payload.MessageID,
			Error:     err.Error(),
		}
		return result, classifySendError(err)
	}

	return &SendEmailResult{
		MessageID:         msg.ID,
		ProviderMessageID: msg.ProviderMessageID,
		Status:            msg.Status,
	}, nil
}

func classifySendError(err error) error {
	if errors.Is(err, services.ErrRetryableEmailSend) {
		return temporaltype.NewRetryableError("Email provider temporarily failed", err).ToTemporalError()
	}
	if errors.Is(err, services.ErrNonRetryableEmailSend) {
		return temporaltype.NewNonRetryableError("Email provider rejected the message", err).ToTemporalError()
	}
	return temporaltype.ClassifyError(err).ToTemporalError()
}
