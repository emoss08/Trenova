package smsjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/sms"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	SMSClient *sms.Client
	Logger    *zap.Logger
}

type Activities struct {
	smsClient *sms.Client
	logger    *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		smsClient: p.SMSClient,
		logger:    p.Logger.Named("sms-activities"),
	}
}

func (a *Activities) SendSMSActivity(
	ctx context.Context,
	payload *SendSMSPayload,
) (*SendSMSResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting SMS send activity",
		"organizationId", payload.OrganizationID.String(),
		"businessUnitId", payload.BusinessUnitID.String(),
	)

	activity.RecordHeartbeat(ctx, "sending SMS")

	err := a.smsClient.Send(sms.SendRequest{
		To:   payload.PhoneNumber,
		Body: payload.Message,
	})
	if err != nil {
		logger.Error("Failed to send SMS", "error", err)
		return &SendSMSResult{
			Success: false,
			Error:   err.Error(),
		}, temporaltype.NewRetryableError("Failed to send SMS", err).ToTemporalError()
	}

	logger.Info("SMS sent successfully")

	return &SendSMSResult{
		Success: true,
	}, nil
}
